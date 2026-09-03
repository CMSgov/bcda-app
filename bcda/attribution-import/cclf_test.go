package attributionimport

import (
	"archive/zip"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CCLFTestSuite struct {
	suite.Suite
	pendingDeletionDir string
	basePath           string
	origDate           string
	importer           CCLFImporter
	cleanup            func()
	db                 *sql.DB
	pool               *pgxv5Pool.Pool
}

// type mockS3ForCleanup struct {
// 	bcdaaws.MockS3Client
// 	deleted map[string]bool
// }

// func newMockS3ForCleanup() *mockS3ForCleanup {
// 	return &mockS3ForCleanup{
// 		deleted: make(map[string]bool),
// 	}
// }

// func (m *mockS3ForCleanup) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
// 	if input != nil && input.Key != nil {
// 		m.deleted[*input.Key] = true
// 	}
// 	return &s3.DeleteObjectOutput{}, nil
// }

// func (m *mockS3ForCleanup) HeadObject(ctx context.Context, input *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
// 	if input != nil && input.Key != nil && m.deleted[*input.Key] {
// 		return nil, &s3types.NotFound{}
// 	}
// 	return &s3.HeadObjectOutput{}, nil
// }

func (s *CCLFTestSuite) SetupSuite() {
	s.origDate = conf.GetEnv("CCLF_REF_DATE")

	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		s.FailNow(err.Error())
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(&s.Suite, dir)

	s.db = database.Connect()
	s.pool = database.ConnectPool()
}

func (s *CCLFTestSuite) SetupTest() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "181201")

	s.basePath, s.cleanup = testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/")

	logger := testUtils.GetLogger(log.API)
	client := bcdaaws.MockS3Client{}
	s.importer = NewCCLFImporter(logger, &client, s.pool)
}

func (s *CCLFTestSuite) TearDownSuite() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", s.origDate)
	os.RemoveAll(s.pendingDeletionDir)
}

func (s *CCLFTestSuite) TearDownTest() {
	s.cleanup()
}

func TestCCLFTestSuite(t *testing.T) {
	suite.Run(t, new(CCLFTestSuite))
}

func (s *CCLFTestSuite) TestImportCCLF0() {
	assert := assert.New(s.T())

	cclfZipfilePath := filepath.Join(s.basePath, "cclf/archives/valid/T.BCD.A0001.ZCY18.D181120.T1000000")
	metadata, zipCloser1 := buildZipMetadata(s.T(), s.importer, "A0001", cclfZipfilePath, "T.BCD.A0001.ZC0Y18.D181120.T1000011", "", models.FileTypeDefault)
	defer zipCloser1()

	// positive
	validator, err := s.importer.importCCLF0(metadata)
	assert.Nil(err)
	assert.Equal(&cclfFileValidator{totalRecordCount: 7, maxRecordLength: 549}, validator)

	// missing cclf8 from cclf0
	cclfZipfilePath = filepath.Join(s.basePath, "cclf/archives/0/missing_data/T.BCD.A0001.ZCY18.D181120.T1000000")
	metadata, zipCloser2 := buildZipMetadata(s.T(), s.importer, "A0001", cclfZipfilePath, "T.BCD.A0001.ZC0Y18.D181120.T1000011", "", models.FileTypeDefault)
	defer zipCloser2()

	_, err = s.importer.importCCLF0(metadata)
	assert.EqualError(err, "failed to parse CCLF8 from CCLF0 file T.BCD.A0001.ZC0Y18.D181120.T1000011")

	// duplicate file types from cclf0
	cclfZipfilePath = filepath.Join(s.basePath, "cclf/archives/0/missing_data/T.BCD.A0001.ZCY18.D181122.T1000000")
	metadata, zipCloser3 := buildZipMetadata(s.T(), s.importer, "A0001", cclfZipfilePath, "T.BCD.A0001.ZC0Y18.D181120.T1000013", "", models.FileTypeDefault)
	defer zipCloser3()

	_, err = s.importer.importCCLF0(metadata)
	assert.EqualError(err, "duplicate CCLF8 file type found from CCLF0 file")

	//invalid record count
	cclfZipfilePath = filepath.Join(s.basePath, "cclf/archives/0/invalid/T.A0001.ACO.ZC0Y18.D181120.Z1000000")
	metadata, zipCloser4 := buildZipMetadata(s.T(), s.importer, "A0001", cclfZipfilePath, "T.A0001.ACO.ZC0Y18.D181120.Z1000011", "", models.FileTypeDefault)
	defer zipCloser4()

	_, err = s.importer.importCCLF0(metadata)
	assert.EqualError(err, "failed to parse CCLF8 record count from CCLF0 file: strconv.Atoi: parsing \"N\": invalid syntax")

	//invalid record length
	cclfZipfilePath = filepath.Join(s.basePath, "cclf/archives/0/invalid/T.BCD.ACOB.ZC0Y18.D181120.E0001000")
	metadata, zipCloser5 := buildZipMetadata(s.T(), s.importer, "A0001", cclfZipfilePath, "T.A0001.ACO.ZC0Y18.D181120.E1000011", "", models.FileTypeDefault)
	defer zipCloser5()

	_, err = s.importer.importCCLF0(metadata)
	assert.EqualError(err, "failed to parse CCLF8 record length from CCLF0 file: strconv.Atoi: parsing \"Num\": invalid syntax")
}

// func (s *CCLFTestSuite) TestImportCCLFDirectoryValid() {
// 	assert := assert.New(s.T())
// 	//Happy case, with directory containing valid BCD files.
// 	_, _, _, err := s.importer.ImportCCLFDirectory(filepath.Join(s.basePath, constants.CCLFDIR, "archives", "valid"))
// 	assert.Nil(err)
// }

func (s *CCLFTestSuite) TestImportCCLFDirectoryInvalid() {
	assert := assert.New(s.T())
	//Directory with mixed file types + at least one bad file.
	cclfDirectory := filepath.Join(s.basePath, constants.CCLFDIR)
	_, _, _, err := s.importer.ImportCCLFDirectory(s.T().Context(), cclfDirectory)
	assert.EqualError(err, "Failed to import 15 files")

	//Target bad file directory
	cclfDirectory = filepath.Join(s.basePath, constants.CCLFDIR, "archives", "invalid_bcd")
	imported, failed, skipped, _ := s.importer.ImportCCLFDirectory(s.T().Context(), cclfDirectory)
	assert.Equal(0, imported)
	assert.Equal(4, failed)
	assert.Equal(0, skipped)
}

func (s *CCLFTestSuite) TestImportCCLFDirectoryTwoLevels() {
	assert := assert.New(s.T())
	//Zero CCLF files in directory
	//additional invalid directory
	cclfDirectory := filepath.Join(s.basePath, constants.CCLFDIR, "emptydir", "archives")
	_, _, _, err := s.importer.ImportCCLFDirectory(s.T().Context(), cclfDirectory)
	assert.EqualError(err, "error in sorting cclf file: nil,: lstat "+cclfDirectory+": no such file or directory")

}

func (s *CCLFTestSuite) TestImportCCLF8() {
	assert := assert.New(s.T())
	ctx := context.Background()

	//indeterminate test results without deletion of both.
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0002")
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0002")

	acoID := "A0001"
	fileTime, _ := time.Parse(time.RFC3339, constants.TestFileTime)

	metadata, zipCloser := buildZipMetadata(s.T(), s.importer, acoID, filepath.Join(s.basePath, constants.CCLF8CompPath), "", constants.CCLF8Name, models.FileTypeDefault)
	metadata.cclf8Metadata.timestamp = fileTime
	defer zipCloser()

	// validation error -- records too long
	validator := cclfFileValidator{
		maxRecordLength:  2,
		totalRecordCount: 7,
	}

	err := s.importer.importCCLF8(ctx, metadata, validator)
	s.ErrorContains(err, "incorrect record length for file (expected: 2, actual: 549)")

	// validation error -- records too long
	validator.maxRecordLength = 549
	validator.totalRecordCount = 2

	err = s.importer.importCCLF8(ctx, metadata, validator)
	s.ErrorContains(err, "unexpected number of records imported for file T.BCD.A0001.ZC8Y18.D181120.T1000009 (expected: 2, actual: 7)")

	// successful
	validator.maxRecordLength = 549
	validator.totalRecordCount = 7

	err = s.importer.importCCLF8(ctx, metadata, validator)
	s.NoError(err)

	// Check if file was created before trying to access it
	files := postgrestest.GetCCLFFilesByName(s.T(), s.db, metadata.cclf8Metadata.name)
	s.NotEmpty(files, "CCLF file should be created in database")

	file := files[0]
	assert.Equal(constants.CCLF8Name, file.Name)
	assert.Equal(acoID, file.ACOCMSID)

	// Normalize timezone to allow us to check for equality
	assert.Equal(fileTime.UTC().Format("010203040506"), file.Timestamp.UTC().Format("010203040506"))
	assert.Equal(20, file.PerformanceYear)
	assert.Equal(constants.ImportComplete, file.ImportStatus)

	mbis, err := postgres.NewRepository(s.db).GetCCLFBeneficiaryMBIs(ctx, file.ID)
	assert.NoError(err)

	assert.Len(mbis, 6)
	sort.Strings(mbis)
	assert.Equal("1A69B98CD30", mbis[0])
	assert.Equal("1A69B98CD31", mbis[1])
	assert.Equal("1A69B98CD32", mbis[2])
	assert.Equal("1A69B98CD33", mbis[3])
	assert.Equal("1A69B98CD34", mbis[4])
	assert.Equal("1A69B98CD35", mbis[5])
}

func (s *CCLFTestSuite) TestImportCCLF8DBErrors() {
	assert := assert.New(s.T())

	//indeterminate test results without deletion of both.
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")
	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A9999")
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A9999")

	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0002")

	metadata, zipCloser := buildZipMetadata(s.T(), s.importer, "A0001", filepath.Join(s.basePath, constants.CCLF8CompPath), "", constants.CCLF8Name, models.FileTypeDefault)
	defer zipCloser()

	validator := cclfFileValidator{
		maxRecordLength:  549,
		totalRecordCount: 1,
	}

	//Send an invalid context to fail DB check
	ctx, function := context.WithCancel(context.Background())
	function()

	err := s.importer.importCCLF8(ctx, metadata, validator)
	assert.EqualError(err, "failed to start pgx transaction: context canceled")
}

func (s *CCLFTestSuite) TestImportCCLF8_alreadyExists() {
	assert := assert.New(s.T())

	hook := test.NewLocal(testUtils.GetLogger(log.API))

	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, "A0001")

	acoID := "A0001"
	cclfFile := &models.CCLFFile{CCLFNum: 8, ACOCMSID: acoID, Timestamp: time.Now(), PerformanceYear: 18, Name: constants.CCLF8Name}
	postgrestest.CreateCCLFFile(s.T(), s.db, cclfFile)

	metadata, zipCloser := buildZipMetadata(s.T(), s.importer, "A0001", filepath.Join(s.basePath, constants.CCLF8CompPath), "", cclfFile.Name, cclfFile.Type)
	defer zipCloser()

	validator := cclfFileValidator{
		maxRecordLength:  549,
		totalRecordCount: 1,
	}

	err := s.importer.importCCLF8(context.Background(), metadata, validator)
	if err != nil {
		s.FailNow("importCCLF8() error: %s", err.Error())
	}

	var exists bool
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, "already exists in database, skipping import...") {
			exists = true
		}
	}

	assert.True(exists, "CCLF8 file should already exist and should not be imported again.")
}

func (s *CCLFTestSuite) TestImportCCLF8_Invalid() {
	assert := assert.New(s.T())

	// since we do not have the correct number of characters, the import should fail.
	fileName, cclfName := createTemporaryCCLF8ZipFile(s.T(), "A 1")
	defer os.Remove(fileName)

	metadata, zipCloser := buildZipMetadata(s.T(), s.importer, "1234", fileName, "", cclfName, models.FileTypeDefault)
	defer zipCloser()

	validator := cclfFileValidator{
		maxRecordLength:  3,
		totalRecordCount: 1,
	}

	err := s.importer.importCCLF8(context.Background(), metadata, validator)
	// This error indicates that we did not supply enough characters for the MBI
	assert.Contains(err.Error(), "invalid byte sequence for encoding \"UTF8\": 0x00")
}

func (s *CCLFTestSuite) TestImportRunoutCCLF() {
	db := s.db

	cmsID := "RNOUT"
	defer func() {
		postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, cmsID)
	}()

	tests := []struct {
		name     string
		fileType models.CCLFFileType
	}{
		{"Default file type", models.FileTypeDefault},
		{"Runout file type", models.FileTypeRunout},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			mbi := "123456789AB" // We expect 11 characters for the MBI

			fileName, cclfName := createTemporaryCCLF8ZipFile(s.T(), mbi)
			defer os.Remove(fileName)

			metadata, zipCloser := buildZipMetadata(s.T(), s.importer, "1234", fileName, "", cclfName, tt.fileType)
			defer zipCloser()

			validator := cclfFileValidator{
				maxRecordLength:  11,
				totalRecordCount: 1,
			}

			s.NoError(s.importer.importCCLF8(context.Background(), metadata, validator))

			cclfFile := postgrestest.GetCCLFFilesByName(s.T(), db, cclfName)[0]
			assert.Equal(t, tt.fileType, cclfFile.Type)
		})
	}
}

func createTemporaryCCLF8ZipFile(t *testing.T, data string) (fileName, cclfName string) {
	cclfName = uuid.New()

	f, err := os.CreateTemp("", "*")
	assert.NoError(t, err)

	w := zip.NewWriter(f)
	f1, err := w.Create(cclfName)
	assert.NoError(t, err)

	_, err = f1.Write([]byte(data))
	assert.NoError(t, err)

	assert.NoError(t, w.Close())

	return f.Name(), cclfName
}

func buildZipMetadata(t *testing.T, importer CCLFImporter, cmsID, zipName, cclf0Name, cclf8Name string, fileType models.CCLFFileType) (*cclfZipMetadata, func()) {

	zipReader, zipCloser, err := importer.openZipArchive(context.Background(), zipName)
	assert.NoError(t, err)

	metadata := cclfZipMetadata{
		filePath:  zipName,
		zipReader: zipReader,
		zipCloser: zipCloser,
		cclf0Metadata: cclfFileMetadata{
			cclfNum:   0,
			name:      cclf0Name,
			acoID:     cmsID,
			timestamp: time.Now(),
			perfYear:  20,
			fileType:  fileType,
		},
		cclf8Metadata: cclfFileMetadata{
			cclfNum:   8,
			name:      cclf8Name,
			acoID:     cmsID,
			timestamp: time.Now(),
			perfYear:  20,
			fileType:  fileType,
		},
	}

	if cclf0Name != "" {
		metadata.cclf0File = *testUtils.GetFileFromZip(t, zipReader, cclf0Name)
	}

	if cclf8Name != "" {
		metadata.cclf8File = *testUtils.GetFileFromZip(t, zipReader, cclf8Name)
	}

	return &metadata, zipCloser
}

// some of these integration tests are disabled because they require a local S3 instance to run.
// They are useful for testing the import of CCLF files from S3 but are not part of the normal test suite.

// func (s *CCLFTestSuite) TestLoadCclfFiles() {
// 	ctx := context.Background()
// 	cmsID := "A0001"
// 	tests := []struct {
// 		path            string
// 		numCCLFZipFiles int
// 		skipped         int
// 		failure         int
// 	}{
// 		{"cclf/archives/valid/", 1, 0, 0},
// 		{"cclf/mixed/with_invalid_filenames/", 1, 0, 0},
// 		{"cclf/mixed/0/valid_names/", 0, 0, 3},
// 		{"cclf/archives/8/valid/", 0, 0, 5},
// 		{"cclf/files/9/valid_names/", 0, 0, 0},
// 		{"cclf/mixed/with_folders/", 1, 0, 0},
// 	}

// 	for _, tt := range tests {
// 		s.T().Run(tt.path, func(t *testing.T) {
// 			bucketName := filepath.Join(s.basePath, tt.path)
// 			// bucketName, cleanup := testUtils.CopyToS3(s.T(), filepath.Join(s.basePath, tt.path))
// 			// defer cleanup()

// 			cclfMap, skipped, failure, err := LoadCclfFiles(ctx, s.fileHelper, filepath.Join(bucketName, tt.path))
// 			cclfZipFiles := cclfMap[cmsID]
// 			assert.NoError(t, err)
// 			assert.Equal(t, tt.skipped, skipped)
// 			assert.Equal(t, tt.failure, failure)
// 			assert.Equal(t, tt.numCCLFZipFiles, len(cclfZipFiles))
// 			for _, cclfZipFile := range cclfZipFiles {
// 				assert.Equal(t, 18, cclfZipFile.cclf0Metadata.perfYear)
// 				assert.Equal(t, 18, cclfZipFile.cclf8Metadata.perfYear)
// 				assert.Equal(t, models.FileTypeDefault, cclfZipFile.cclf0Metadata.fileType)
// 				assert.Equal(t, models.FileTypeDefault, cclfZipFile.cclf8Metadata.fileType)
// 			}
// 		})
// 	}
// }

func (s *CCLFTestSuite) TestLoadCclfFiles_SkipOtherEnvs() {
	ctx := context.Background()
	cleanupEnvVars := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{{Name: "ENV", Value: "dev"}})
	s.T().Cleanup(func() { cleanupEnvVars() })

	bucketName := uuid.NewRandom().String()
	// bucketName, cleanupS3 := testUtils.CreateZipsInS3(s.T(), testUtils.ZipInput{ZipName: "blah/not-dev/T.BCD.A0001.ZCY18.D181120.T1000000", CclfNames: []string{"T.BCD.A0001.ZC0Y18.D181120.T1000000", "T.BCD.A0001.ZC8Y18.D181120.T1000000"}})
	// s.T().Cleanup(func() { cleanupS3() })

	cclfMap, skipped, failure, err := s.importer.loadCclfFiles(ctx, bucketName)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 0, failure)
	assert.Empty(s.T(), cclfMap)
}

// func (s *CCLFTestSuite) TestLoadCclfFiles_DuplicateCCLFs() {
// 	ctx := context.Background()
// 	bucketName := uuid.NewString()
// 	// bucketName, cleanupS3 := testUtils.CreateZipsInS3(s.T(),
// 	// 	// Multiple CCLF0s
// 	// 	testUtils.ZipInput{
// 	// 		ZipName:   "T.BCD.A9990.ZCY20.D201113.T0000000",
// 	// 		CclfNames: []string{"T.BCD.A9990.ZC0Y20.D201113.T0000010", "T.BCD.A9990.ZC0Y20.D201113.T0000011", "T.BCD.A9990.ZC8Y20.D201113.T0000010"},
// 	// 	},
// 	// 	// Multiple CCLF8s
// 	// 	testUtils.ZipInput{
// 	// 		ZipName:   "T.BCD.A9990.ZCY19.D201113.T0000000",
// 	// 		CclfNames: []string{"T.BCD.A9990.ZC0Y19.D201113.T0000010", "T.BCD.A9990.ZC8Y19.D201113.T0000010", "T.BCD.A9990.ZC8Y19.D201113.T0000011"},
// 	// 	},
// 	// )
// 	// defer cleanupS3()

// 	cclfMap, skipped, failure, err := LoadCclfFiles(ctx, s.fileHelper, bucketName)
// 	assert.Nil(s.T(), err)
// 	assert.Equal(s.T(), 0, skipped)
// 	assert.Equal(s.T(), 2, failure)
// 	assert.Empty(s.T(), cclfMap)
// }

// func (s *CCLFTestSuite) TestLoadCclfFiles_SingleFile() {
// 	ctx := context.Background()
// 	cmsID := "A0001"
// 	tests := []struct {
// 		path            string
// 		filename        string
// 		numCCLFZipFiles int // Expected count for the cmsID, perfYear above
// 		skipped         int
// 		failure         int
// 	}{
// 		{"cclf/archives/valid/", "T.BCD.A0001.ZCY18.D181120.T1000000", 1, 0, 0},
// 	}

// 	for _, tt := range tests {
// 		s.T().Run(tt.path, func(t *testing.T) {
// 			bucketName := uuid.NewString()
// 			// bucketName, cleanup := testUtils.CopyToS3(s.T(), filepath.Join(s.basePath, tt.path))
// 			// defer cleanup()

// 			cclfMap, skipped, failure, err := LoadCclfFiles(ctx, s.fileHelper, filepath.Join(bucketName, tt.path, tt.filename))
// 			cclfZipFiles := cclfMap[cmsID]
// 			assert.NoError(t, err)
// 			assert.Equal(t, tt.skipped, skipped)
// 			assert.Equal(t, tt.failure, failure)
// 			assert.Equal(t, tt.numCCLFZipFiles, len(cclfZipFiles))

// 			for _, cclfZipFile := range cclfZipFiles {
// 				assert.Equal(t, 18, cclfZipFile.cclf0Metadata.perfYear)
// 				assert.Equal(t, 18, cclfZipFile.cclf8Metadata.perfYear)
// 				assert.Equal(t, models.FileTypeDefault, cclfZipFile.cclf0Metadata.fileType)
// 				assert.Equal(t, models.FileTypeDefault, cclfZipFile.cclf8Metadata.fileType)
// 			}
// 		})
// 	}
// }

// func (s *CCLFTestSuite) TestLoadCclfFiles_InvalidPath() {
// 	ctx := context.Background()
// 	cclfMap, skipped, failure, err := LoadCclfFiles(ctx, s.fileHelper, "foo")
// 	assert.ErrorContains(s.T(), err, "NoSuchBucket")
// 	assert.Equal(s.T(), 0, skipped)
// 	assert.Equal(s.T(), 0, failure)
// 	assert.Empty(s.T(), cclfMap)
// }

// func (s *CCLFTestSuite) TestMultipleFileTypes() {
// 	ctx := context.Background()
// 	// Hard code the reference date to ensure we do not reject any CCLF files because they are too old.
// 	origDate := conf.GetEnv("CCLF_REF_DATE")
// 	conf.SetEnv(s.T(), "CCLF_REF_DATE", "201201")
// 	s.T().Cleanup(func() { conf.SetEnv(s.T(), "CCLF_REF_DATE", origDate) })
// 	bucketName := uuid.NewString()

// 	// Create various CCLF files that have unique perfYear:fileType
// 	// bucketName, cleanup := testUtils.CreateZipsInS3(s.T(),
// 	// 	testUtils.ZipInput{
// 	// 		ZipName:   "T.BCD.A9990.ZCY20.D201113.T0000000",
// 	// 		CclfNames: []string{"T.BCD.A9990.ZC0Y20.D201113.T0000010", "T.BCD.A9990.ZC8Y20.D201113.T0000010"},
// 	// 	},
// 	// 	// different perf year
// 	// 	testUtils.ZipInput{
// 	// 		ZipName:   "T.BCD.A9990.ZCY19.D201113.T0000000",
// 	// 		CclfNames: []string{"T.BCD.A9990.ZC0Y19.D201113.T0000010", "T.BCD.A9990.ZC8Y19.D201113.T0000010"},
// 	// 	},
// 	// 	// different file type
// 	// 	testUtils.ZipInput{
// 	// 		ZipName:   "T.BCD.A9990.ZCR20.D201113.T0000000",
// 	// 		CclfNames: []string{"T.BCD.A9990.ZC0R20.D201113.T0000010", "T.BCD.A9990.ZC8R20.D201113.T0000010"},
// 	// 	},
// 	// 	// different perf year and file type
// 	// 	testUtils.ZipInput{
// 	// 		ZipName:   "T.BCD.A9990.ZCR19.D201113.T0000000",
// 	// 		CclfNames: []string{"T.BCD.A9990.ZC0R19.D201113.T0000010", "T.BCD.A9990.ZC8R19.D201113.T0000010"},
// 	// 	},
// 	// )
// 	// s.T().Cleanup(func() { cleanup() })

// 	m, skipped, f, err := LoadCclfFiles(ctx, s.fileHelper, bucketName)
// 	assert.NoError(s.T(), err)
// 	assert.Equal(s.T(), 0, skipped)
// 	assert.Equal(s.T(), 0, f)
// 	assert.Equal(s.T(), 1, len(m)) // Only one ACO present

// 	for _, fileMap := range m {
// 		// We should contain 4 unique entries, one for each unique perfYear:fileType tuple
// 		assert.Equal(s.T(), 4, len(fileMap))
// 	}
// }

func (s *CCLFTestSuite) TestCleanupCCLF() {
	assert := assert.New(s.T())
	cclfmap := make(map[string][]*cclfZipMetadata)
	acoID := "A0001"

	bucketName := uuid.NewRandom().String()

	// failed import: stay put
	fileTime, _ := time.Parse(time.RFC3339, constants.TestFileTime)
	cclf0metadata := &cclfFileMetadata{
		name:         "T.BCD.ACO.ZC0Y18.D181120.T0001000",
		env:          "test",
		acoID:        acoID,
		cclfNum:      8,
		perfYear:     18,
		timestamp:    fileTime,
		deliveryDate: time.Now(),
	}

	fileTime, _ = time.Parse(time.RFC3339, constants.TestFileTime)
	cclf8metadata := &cclfFileMetadata{
		name:         constants.CCLF8Name,
		env:          "test",
		acoID:        acoID,
		cclfNum:      8,
		perfYear:     18,
		timestamp:    fileTime,
		deliveryDate: time.Now(),
	}

	cclfmap[acoID] = []*cclfZipMetadata{
		{
			cclf0Metadata: *cclf0metadata,
			cclf8Metadata: *cclf8metadata,
			filePath:      filepath.Join(bucketName, constants.CCLF8CompPath),
			imported:      false,
		},
	}

	deletedCount, err := s.importer.cleanUpCCLF(context.Background(), cclfmap)
	assert.Equal(0, deletedCount)
	assert.Nil(err)

	// Cleanup file after import
	cclfmap[acoID][0].imported = true

	deletedCount, err = s.importer.cleanUpCCLF(context.Background(), cclfmap)
	assert.Equal(1, deletedCount)
	assert.Nil(err)
}
