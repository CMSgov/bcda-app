package cclf

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database/databasetest"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/CMSgov/bcda-app/optout"
	"github.com/ccoveille/go-safecast"
	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CSVTestSuite struct {
	suite.Suite
	basePath string
	importer CSVImporter
	cleanup  func()
	db       *sql.DB
}

func (s *CSVTestSuite) SetupTest() {
	s.basePath, s.cleanup = testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/")

	db, _ := databasetest.CreateDatabase(s.T(), "../../db/migrations/bcda/", true)
	tf, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect("postgres"),
	)
	if err != nil {
		assert.FailNowf(s.T(), "Failed to setup test fixtures", err.Error())
	}
	if err = tf.Load(); err != nil {
		assert.FailNowf(s.T(), "Failed to load test fixtures", err.Error())
	}
	s.db = db
	hours, err := safecast.ToUint(utils.GetEnvInt("FILE_ARCHIVE_THRESHOLD_HR", 72))
	if err != nil {
		fmt.Println("Error converting FILE_ARCHIVE_THRESHOLD_HR to uint", err)
	}
	fp := &LocalFileProcessor{
		Handler: optout.LocalFileHandler{
			Logger:                 log.API,
			PendingDeletionDir:     conf.GetEnv("PENDING_DELETION_DIR"),
			FileArchiveThresholdHr: hours,
		}}

	c := CSVImporter{
		Logger:        log.API,
		FileProcessor: fp,
		Database:      s.db,
	}
	s.importer = c

}
func (s *CSVTestSuite) SetupSuite() {
}

func (s *CSVTestSuite) TearDownSuite() {

}

func (s *CSVTestSuite) TearDownTest() {
	s.cleanup()
}

func TestCSVTestSuite(t *testing.T) {
	suite.Run(t, new(CSVTestSuite))
}

func (s *CSVTestSuite) TestImportCSV_Integration() {
	// TODO apply fixture data for dupe file test case
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "181201")
	tests := []struct {
		name        string
		filepath    string
		cclfFileID  int
		cclfBeneRec []string
		err         error
	}{
		{"Import CSV attribution success", filepath.Join(s.basePath, "cclf/archives/csv/P.PCPB.M2411.D181120.T1000000"), 0, []string{"MBI000001", "MBI000002", "MBI000003", "MBI000004", "MBI000005"}, nil},
		//{"Import CSV attribution that already exists", "", 0, []string{}},
		{"Import CSV attribution invalid name", filepath.Join(s.basePath, "cclf/archives/csv/P.PC.M2411.D181120.T1000000"), 0, []string{}, errors.New("invalid filename")},
		{"Import Opt Out failure", filepath.Join(s.basePath, "cclf/archives/csv/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000010"), 0, []string{}, errors.New("invalid filename")},
	}

	for _, test := range tests {
		s.T().Run(test.name, func(tt *testing.T) {
			filename := filepath.Clean(test.filepath)
			err := s.importer.ImportCSV(test.filepath)
			if test.err == nil {
				assert.Nil(s.T(), err)
			} else {
				assert.NotNil(s.T(), err)
				assert.Contains(s.T(), err.Error(), test.err.Error())
			}
			cclfRecords := postgrestest.GetCCLFFilesByName(tt, s.db, filepath.Clean(test.filepath))
			if len(cclfRecords) != 0 {
				assert.Equal(tt, 1, len(cclfRecords))
				assert.Equal(tt, filename, cclfRecords[0].Name)
				//assert.NotNil()
				beneRecords, _ := postgrestest.GetCCLFBeneficiaries(s.db, int(cclfRecords[0].ID))
				assert.Equal(s.T(), len(test.cclfBeneRec), len(beneRecords))
				for i, v := range beneRecords {
					fmt.Println(i, v)
					assert.Contains(s.T(), test.cclfBeneRec, (strings.ReplaceAll(v, " ", "")))
				}
			} else {
				assert.Equal(tt, 0, len(cclfRecords))
			}
		})
	}

}

func (s *CSVTestSuite) TestProcessCSV_Integration() {

	file := csvFile{
		metadata: csvFileMetadata{
			name:         "P.PCPB.M2411.D191005.T0209260",
			env:          "test",
			acoID:        "FOOACO",
			cclfNum:      8,
			perfYear:     24,
			timestamp:    time.Now(),
			deliveryDate: time.Now(),
			fileID:       0,
			fileType:     1,
		},
		data: bytes.NewReader([]byte("MBIS\nMBI000001\nMBI000002\nMBI000003")),
	}

	expectedFile := models.CCLFFile{
		Name:            "P.PCPB.M2411.D191005.T0209260",
		ACOCMSID:        "FOOACO",
		PerformanceYear: 24,
	}
	tests := []struct {
		name       string
		file       csvFile
		fileRecord models.CCLFFile
		mbiRecord  []string
		err        error
	}{
		{"Import CSV attribution success", file, expectedFile, []string{"MBI000001", "MBI000002", "MBI000003"}, nil},
		{"Import CSV attribution that already exists", file, models.CCLFFile{}, []string{}, errors.New("already exists")},
	}

	for _, test := range tests {
		s.T().Run(test.name, func(tt *testing.T) {
			err := s.importer.ProcessCSV(file)
			if test.err != nil {
				err := s.importer.ProcessCSV(file)
				assert.NotNil(s.T(), err)
				assert.Contains(s.T(), err.Error(), test.err.Error())
			} else {
				assert.Nil(tt, err)
				cclfRecord := postgrestest.GetCCLFFilesByName(s.T(), s.db, file.metadata.name)
				assert.Equal(s.T(), 1, len(cclfRecord))
				assert.Equal(s.T(), expectedFile.Name, cclfRecord[0].Name)
				assert.Equal(s.T(), expectedFile.ACOCMSID, cclfRecord[0].ACOCMSID)
				assert.Equal(s.T(), expectedFile.PerformanceYear, cclfRecord[0].PerformanceYear)
				assert.Equal(s.T(), constants.ImportComplete, cclfRecord[0].ImportStatus)

				beneRecords, _ := postgres.NewRepository(s.db).GetCCLFBeneficiaryMBIs(context.Background(), cclfRecord[0].ID)
				sort.Strings(beneRecords)
				assert.Equal(s.T(), 3, len(beneRecords))
				for i, v := range beneRecords {
					fmt.Println(i, v)
					assert.Contains(s.T(), []string{"MBI000001", "MBI000002", "MBI000003"}, (strings.ReplaceAll(v, " ", "")))
				}
			}

		})
	}
}

// func (s *CSVTestSuite) TestProcessCSV() {

// 	csvFile := csvFile{
// 		metadata: cclfFileMetadata{
// 			name:         "P.PCPB.M2411.D191005.T0209260",
// 			env:          "test",
// 			acoID:        "FOOACO",
// 			cclfNum:      8,
// 			perfYear:     24,
// 			timestamp:    time.Now(),
// 			deliveryDate: time.Now(),
// 			fileID:       0,
// 			fileType:     1,
// 		},
// 		fileContent: bytes.NewReader([]byte("MBIS\nMBI000001\nMBI000002\nMBI000003")),
// 	}

// 	repository := &models.MockRepository{}

// 	tests := []struct {
// 		name string
// 		data *mock.Call
// 		err  string
// 	}{
// 		{"GetCCLFFileExistsByName", repository.On("GetLatestCCLFFile").Return("", errors.New("failed to check existence")), "failed to check existence"},
// 	}
// 	for _, test := range tests {
// 		s.T().Run(test.name, func(tt *testing.T) {
// 			err := s.importer.ProcessCSV(csvFile)
// 			assert.Nil(tt, err)
// 		})
// 	}

//}

func TestPrepareCSVData(t *testing.T) {
	c := CSVImporter{}
	tests := []struct {
		name     string
		data     *bytes.Reader
		err      error
		expected [][]interface{}
	}{
		{"Valid CSV file with content", bytes.NewReader([]byte("MBIS\nMBI000001\nMBI000002\nMBI000003")), nil, [][]interface{}{
			{uint(1), "MBI000001"},
			{uint(1), "MBI000002"},
			{uint(1), "MBI000003"},
		}},
		{"Empty CSV file", bytes.NewReader([]byte("")), io.EOF, [][]interface{}(nil)},
		{"Valid CSV file with unexpected content", bytes.NewReader([]byte("MBIS\nMBI000001,10\nMBI000002,bar\nMBI000003,")), nil, [][]interface{}{
			{uint(1), "MBI000001"},
			{uint(1), "MBI000002"},
			{uint(1), "MBI000003"},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fmt.Print(test.name)
			rows, _, err := c.prepareCSVData(test.data, uint(1))
			assert.Equal(t, test.expected, rows)
			if err != nil {
				assert.Equal(t, test.err.Error(), err.Error())
			}
		})
	}

}
