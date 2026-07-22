package attributionimport

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	bp "github.com/CMSgov/bcda-app/bcda/bene-prefs"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/db"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type CSVTestSuite struct {
	suite.Suite
	basePath           string
	importer           CSVImporter
	cleanup            func()
	db                 *sql.DB
	pool               *pgxv5Pool.Pool
	origDate           string
	pendingDeletionDir string
	dbContainer        db.TestDatabaseContainer
}

func (s *CSVTestSuite) SetupSuite() {
	var err error
	s.origDate = conf.GetEnv("CCLF_REF_DATE")
	s.dbContainer, err = db.NewTestDatabaseContainer()
	require.NoError(s.T(), err)
}

func (s *CSVTestSuite) TearDownSuite() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", s.origDate)
	os.RemoveAll(conf.GetEnv("PENDING_DELETION_DIR"))
	defer func() {
		if err := testcontainers.TerminateContainer(s.dbContainer.Container); err != nil {
			s.T().Log(fmt.Errorf("failed to terminate container: %w", err))
		}
	}()
}

func (s *CSVTestSuite) SetupTest() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "181201")
	s.basePath, s.cleanup = testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/")
	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		s.FailNow(err.Error())
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(&s.Suite, dir)
	hours, err := safecast.ToUint(utils.GetEnvInt("FILE_ARCHIVE_THRESHOLD_HR", 72))
	if err != nil {
		fmt.Println("Error converting FILE_ARCHIVE_THRESHOLD_HR to uint", err)
	}
	fp := &LocalFileProcessor{
		Handler: bp.LocalFileHandler{
			Logger:                 log.API,
			PendingDeletionDir:     conf.GetEnv("PENDING_DELETION_DIR"),
			FileArchiveThresholdHr: hours,
		}}

	c := CSVImporter{
		Logger:        log.API,
		FileProcessor: fp,
	}
	s.importer = c

}

func (s *CSVTestSuite) TearDownTest() {
	s.cleanup()

}

func (s *CSVTestSuite) SetupSubTest() {
	var err error
	s.db, err = s.dbContainer.NewSqlDbConnection()
	require.NoError(s.T(), err)
	s.pool, err = s.dbContainer.NewPgxPoolConnection()
	require.NoError(s.T(), err)
	s.importer.PgxPool = s.pool
}

func (s *CSVTestSuite) TearDownSubTest() {
	s.db.Close()
	err := s.dbContainer.RestoreSnapshot("Base")
	require.NoError(s.T(), err)
}

func TestCSVTestSuite(t *testing.T) {
	suite.Run(t, new(CSVTestSuite))
}

func (s *CSVTestSuite) TestImportCSV_Integration() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "181201")
	tests := []struct {
		name        string
		filepath    string
		cclfFileID  int
		cclfBeneRec []string
		err         error
	}{
		{"Import CSV attribution success", filepath.Join(s.basePath, "cclf/archives/csv/P.PCPB.M2411.D181120.T1000000"), 0, []string{"MBI000001", "MBI000002", "MBI000003", "MBI000004", "MBI000005"}, nil},
		{"Import CSV attribution that already exists", filepath.Join(s.basePath, "cclf/archives/csv/P.PCPB.M2411.D181121.T1000000"), 0, []string{}, errors.New("already exists")},
		{"Import CSV attribution invalid name", filepath.Join(s.basePath, "cclf/archives/csv/P.PC.M2411.D181120.T1000000"), 0, []string{}, errors.New("Invalid filename")},
		{"Import bene-prefs failure", filepath.Join(s.basePath, "cclf/archives/csv/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000010"), 0, []string{}, errors.New("File is type: bene-prefs. Skipping attribution import.")},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			err := s.dbContainer.ExecuteDir("testdata/")
			require.NoError(s.T(), err)
			filename := filepath.Clean(test.filepath)
			err = s.importer.ImportCSV(context.Background(), test.filepath)
			if test.err == nil {
				assert.Nil(s.T(), err)
			} else {
				assert.NotNil(s.T(), err)
				assert.Contains(s.T(), err.Error(), test.err.Error())
			}
			r := postgres.NewRepository(s.db)
			cclfRecords := postgrestest.GetCCLFFilesByName(s.T(), s.db, filepath.Clean(test.filepath))
			if len(cclfRecords) != 0 {
				assert.Equal(s.T(), 1, len(cclfRecords))
				assert.Equal(s.T(), filename, cclfRecords[0].Name)
				beneRecords, _ := r.GetCCLFBeneficiaries(context.Background(), cclfRecords[0].ID, []string{})
				assert.Equal(s.T(), len(test.cclfBeneRec), len(beneRecords))
				for _, v := range beneRecords {
					assert.Contains(s.T(), test.cclfBeneRec, (strings.ReplaceAll(v.MBI, " ", "")))
				}
			} else {
				assert.Equal(s.T(), 0, len(cclfRecords))
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

	dupeFile := csvFile{
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
		data: bytes.NewReader([]byte("MBIS\nMBI000004\nMBI000005\nMBI000006")),
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
		{"Import CSV attribution that already exists", dupeFile, expectedFile, []string{"MBI000001", "MBI000002", "MBI000003"}, errors.New("already exists")},
		{"Import CSV attribution success", file, expectedFile, []string{"MBI000001", "MBI000002", "MBI000003"}, nil},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			err := s.importer.ProcessCSV(test.file)
			if test.err != nil {
				cclfRecord := postgrestest.GetCCLFFilesByName(s.T(), s.db, file.metadata.name)
				assert.Equal(s.T(), 1, len(cclfRecord))
				assert.Nil(s.T(), err)
				err = s.importer.ProcessCSV(test.file)
				assert.NotNil(s.T(), err)
				assert.Contains(s.T(), err.Error(), test.err.Error())
			} else {
				assert.Nil(s.T(), err)
				cclfRecord := postgrestest.GetCCLFFilesByName(s.T(), s.db, file.metadata.name)
				assert.Equal(s.T(), 1, len(cclfRecord))
				assert.Nil(s.T(), err)
				assert.Equal(s.T(), expectedFile.Name, cclfRecord[0].Name)
				assert.Equal(s.T(), expectedFile.ACOCMSID, cclfRecord[0].ACOCMSID)
				assert.Equal(s.T(), expectedFile.PerformanceYear, cclfRecord[0].PerformanceYear)
				assert.Equal(s.T(), constants.ImportComplete, cclfRecord[0].ImportStatus)

				beneRecords, _ := postgres.NewRepository(s.db).GetCCLFBeneficiaryMBIs(context.Background(), cclfRecord[0].ID)
				sort.Strings(beneRecords)
				assert.Equal(s.T(), 3, len(beneRecords))
				for _, v := range beneRecords {
					assert.Contains(s.T(), []string{"MBI000001", "MBI000002", "MBI000003"}, (strings.ReplaceAll(v, " ", "")))
				}
			}

		})
	}
}

func (s *CSVTestSuite) TestPrepareCSVData() {

	c := CSVImporter{
		PgxPool: s.pool,
	}
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
		{"Empty CSV file", bytes.NewReader([]byte("")), errors.New("empty attribution file"), [][]interface{}(nil)},
		{"Valid CSV file with unexpected content - more columns than headers", bytes.NewReader([]byte("MBIS\nMBI000001,10\nMBI000002,bar\nMBI000003,")), errors.New("failed to read csv attribution file"), [][]interface{}(nil)},
		{"Valid CSV file with unexpected content - extra column and header", bytes.NewReader([]byte("MBIS,foo\nMBI000001,10\nMBI000002,bar\nMBI000003,")), nil, [][]interface{}{
			{uint(1), "MBI000001"},
			{uint(1), "MBI000002"},
			{uint(1), "MBI000003"},
		}},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			fmt.Print(test.name)
			rows, _, err := c.prepareCSVData(test.data, uint(1))
			assert.Equal(s.T(), test.expected, rows)
			if err != nil {
				assert.Contains(s.T(), err.Error(), test.err.Error())
			}
		})
	}

}
