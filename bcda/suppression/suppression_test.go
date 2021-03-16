package suppression

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SuppressionTestSuite struct {
	suite.Suite
	pendingDeletionDir string

	basePath string
	cleanup  func()
}

func (s *SuppressionTestSuite) SetupSuite() {
	dir, err := ioutil.TempDir("", "*")
	if err != nil {
		log.Fatal(err)
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(s.Suite, dir)
}

func (s *SuppressionTestSuite) SetupTest() {
	s.basePath, s.cleanup = testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/")
}

func (s *SuppressionTestSuite) TearDownSuite() {
	os.RemoveAll(s.pendingDeletionDir)
}

func (s *SuppressionTestSuite) TearDownTest() {
	s.cleanup()
}
func TestSuppressionTestSuite(t *testing.T) {
	suite.Run(t, new(SuppressionTestSuite))
}

func (s *SuppressionTestSuite) TestImportSuppression() {
	assert := assert.New(s.T())
	db := database.Connection
	defer db.Close()

	// 181120 file
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata := &suppressionFileMetadata{
		timestamp:    fileTime,
		filePath:     filepath.Join(s.basePath, "synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"),
		name:         "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009",
		deliveryDate: time.Now(),
	}
	err := importSuppressionData(metadata)
	assert.Nil(err)

	suppressionFile := postgrestest.GetSuppressionFileByName(s.T(), db, metadata.name)[0]
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009", suppressionFile.Name)
	assert.Equal(fileTime.Format("010203040506"), suppressionFile.Timestamp.Format("010203040506"))
	assert.Equal(constants.ImportComplete, suppressionFile.ImportStatus)

	suppressions := postgrestest.GetSuppressionsByFileID(s.T(), db, suppressionFile.ID)
	assert.Len(suppressions, 4)
	assert.Equal("5SJ0A00AA00", suppressions[0].MBI)
	assert.Equal("1-800", suppressions[0].SourceCode)
	assert.Equal("4SF6G00AA00", suppressions[1].MBI)
	assert.Equal("1-800", suppressions[1].SourceCode)
	assert.Equal("4SH0A00AA00", suppressions[2].MBI)
	assert.Equal("", suppressions[2].SourceCode)
	assert.Equal("8SG0A00AA00", suppressions[3].MBI)
	assert.Equal("1-800", suppressions[3].SourceCode)

	postgrestest.DeleteSuppressionFileByID(s.T(), db, suppressionFile.ID)

	// 190816 file T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390
	fileTime, _ = time.Parse(time.RFC3339, "2019-08-16T02:41:39Z")
	metadata = &suppressionFileMetadata{
		timestamp:    fileTime,
		filePath:     filepath.Join(s.basePath, "synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390"),
		name:         "T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390",
		deliveryDate: time.Now(),
	}
	err = importSuppressionData(metadata)
	assert.Nil(err)

	suppressionFile = postgrestest.GetSuppressionFileByName(s.T(), db, metadata.name)[0]
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390", suppressionFile.Name)
	assert.Equal(fileTime.Format("010203040506"), suppressionFile.Timestamp.Format("010203040506"))

	suppressions = postgrestest.GetSuppressionsByFileID(s.T(), db, suppressionFile.ID)
	assert.Len(suppressions, 250)
	assert.Equal("1000000019", suppressions[0].MBI)
	assert.Equal("N", suppressions[0].PrefIndicator)
	assert.Equal("1000039915", suppressions[20].MBI)
	assert.Equal("N", suppressions[20].PrefIndicator)
	assert.Equal("1000099805", suppressions[100].MBI)
	assert.Equal("N", suppressions[100].PrefIndicator)
	assert.Equal("1000026399", suppressions[200].MBI)
	assert.Equal("N", suppressions[200].PrefIndicator)
	assert.Equal("1000098787", suppressions[249].MBI)
	assert.Equal(" ", suppressions[249].PrefIndicator)

	postgrestest.DeleteSuppressionFileByID(s.T(), db, suppressionFile.ID)
}

func (s *SuppressionTestSuite) TestImportSuppression_MissingData() {
	assert := assert.New(s.T())
	db := database.Connection
	defer db.Close()

	// Verify empty file is rejected
	metadata := &suppressionFileMetadata{}
	err := importSuppressionData(metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not read file")

	tests := []struct {
		name   string
		expErr string
	}{
		{"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000011", "failed to parse the effective date '20191301' from file"},
		{"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000012", "failed to parse the samhsa effective date '20191301' from file"},
		{"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000013", "failed to parse beneficiary link key from file"},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			fp := filepath.Join(s.basePath, "suppressionfile_MissingData/"+tt.name)
			metadata = &suppressionFileMetadata{
				timestamp:    time.Now(),
				filePath:     fp,
				name:         tt.name,
				deliveryDate: time.Now(),
			}
			err = importSuppressionData(metadata)
			assert.NotNil(err)
			assert.Contains(err.Error(), fmt.Sprintf("%s: %s", tt.expErr, fp))

			suppressionFile := postgrestest.GetSuppressionFileByName(s.T(), db, metadata.name)[0]
			assert.Equal(constants.ImportFail, suppressionFile.ImportStatus)
			postgrestest.DeleteSuppressionFileByID(s.T(), db, suppressionFile.ID)
		})
	}
}

func (s *SuppressionTestSuite) TestValidate() {
	assert := assert.New(s.T())

	// positive
	suppressionfilePath := filepath.Join(s.basePath, "synthetic1800MedicareFiles/test/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009")
	metadata := &suppressionFileMetadata{timestamp: time.Now(), filePath: suppressionfilePath}
	err := validate(metadata)
	assert.Nil(err)

	// bad file path
	metadata.filePath = metadata.filePath + "/blah/"
	err = validate(metadata)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not read file "+metadata.filePath)

	// invalid file header
	metadata.filePath = filepath.Join(s.basePath, "suppressionfile_BadHeader/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009")
	err = validate(metadata)
	assert.EqualError(err, "invalid file header for file: "+metadata.filePath)

	// missing record count
	metadata.filePath = filepath.Join(s.basePath, "suppressionfile_MissingData/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009")
	err = validate(metadata)
	assert.EqualError(err, "failed to parse record count from file: "+metadata.filePath)

	// incorrect record count
	metadata.filePath = filepath.Join(s.basePath, "suppressionfile_MissingData/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000010")
	err = validate(metadata)
	assert.EqualError(err, "incorrect number of records found from file: '"+metadata.filePath+"'. Expected record count: 5, Actual record count: 4")
}

func (s *SuppressionTestSuite) TestParseMetadata() {
	assert := assert.New(s.T())

	// positive
	expTime, _ := time.Parse(time.RFC3339, "2018-11-20T20:13:01Z")
	metadata, err := parseMetadata("blah/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T2013010")
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D181120.T2013010", metadata.name)
	assert.Equal(expTime.Format("010203040506"), metadata.timestamp.Format("010203040506"))
	assert.Nil(err)

	// change the name and timestamp
	expTime, _ = time.Parse(time.RFC3339, "2019-12-20T21:09:42Z")
	metadata, err = parseMetadata("blah/T#EFT.ON.ACO.NGD1800.DPRF.D191220.T2109420")
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D191220.T2109420", metadata.name)
	assert.Equal(expTime.Format("010203040506"), metadata.timestamp.Format("010203040506"))
	assert.Nil(err)
}

func (s *SuppressionTestSuite) TestParseMetadata_InvalidFilename() {
	assert := assert.New(s.T())

	// invalid file name
	_, err := parseMetadata("/path/to/file")
	assert.EqualError(err, "invalid filename for file: /path/to/file")

	_, err = parseMetadata("/path/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000010")
	assert.EqualError(err, "invalid filename for file: /path/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000010")

	// invalid date
	_, err = parseMetadata("/path/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420")
	assert.EqualError(err, "failed to parse date 'D190117.T990942' from file: /path/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420: parsing time \"D190117.T990942\": hour out of range")
}

func (s *SuppressionTestSuite) TestGetSuppressionFileMetadata() {
	assert := assert.New(s.T())
	var suppresslist []*suppressionFileMetadata
	var skipped int

	filePath := filepath.Join(s.basePath, "synthetic1800MedicareFiles/test/")
	err := filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(2, len(suppresslist))
	assert.Equal(0, skipped)

	suppresslist = []*suppressionFileMetadata{}
	skipped = 0
	filePath = filepath.Join(s.basePath, "suppressionfile_BadFileNames/")
	err = filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(0, len(suppresslist))
	assert.Equal(2, skipped)

	suppresslist = []*suppressionFileMetadata{}
	skipped = 0
	filePath = filepath.Join(s.basePath, "synthetic1800MedicareFiles/test/")
	err = filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	modtimeAfter := time.Now().Truncate(time.Second)
	// check current value and change mod time
	for _, f := range suppresslist {
		fInfo, _ := os.Stat(f.filePath)
		assert.Equal(fInfo.ModTime().Format("010203040506"), f.deliveryDate.Format("010203040506"))

		err = os.Chtimes(f.filePath, modtimeAfter, modtimeAfter)
		if err != nil {
			s.FailNow("Failed to change modified time for file", err)
		}
	}

	suppresslist = []*suppressionFileMetadata{}
	filePath = filepath.Join(s.basePath, "synthetic1800MedicareFiles/test/")
	err = filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	for _, f := range suppresslist {
		assert.Equal(modtimeAfter.Format("010203040506"), f.deliveryDate.Format("010203040506"))
	}
}

func (s *SuppressionTestSuite) TestGetSuppressionFileMetadata_TimeChange() {
	assert := assert.New(s.T())
	var suppresslist []*suppressionFileMetadata
	var skipped int
	folderPath := filepath.Join(s.basePath, "suppressionfile_BadFileNames/")
	filePath := filepath.Join(folderPath, "T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009")

	origTime := time.Now().Truncate(time.Second)
	err := os.Chtimes(filePath, origTime, origTime)
	if err != nil {
		s.FailNow("Failed to change modified time for file", err)
	}

	skipped = 0
	err = filepath.Walk(folderPath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(0, len(suppresslist))
	assert.Equal(2, skipped)

	// assert that this file is still here.
	_, err = os.Open(filePath)
	assert.Nil(err)

	timeChange := origTime.Add(-(time.Hour * 73)).Truncate(time.Second)
	err = os.Chtimes(filePath, timeChange, timeChange)
	if err != nil {
		s.FailNow("Failed to change modified time for file", err)
	}

	suppresslist = []*suppressionFileMetadata{}
	skipped = 0
	err = filepath.Walk(folderPath, getSuppressionFileMetadata(&suppresslist, &skipped))
	assert.Nil(err)
	assert.Equal(0, len(suppresslist))
	assert.Equal(2, skipped)

	// assert that this file is not still here.
	_, err = os.Open(filePath)
	assert.EqualError(err, fmt.Sprintf("open %s: no such file or directory", filePath))
}

func (s *SuppressionTestSuite) TestCleanupSuppression() {
	assert := assert.New(s.T())
	var suppresslist []*suppressionFileMetadata

	// failed import: file that's within the threshold - stay put
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:09Z")
	metadata := &suppressionFileMetadata{
		name:         "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009",
		timestamp:    fileTime,
		filePath:     filepath.Join(s.basePath, "suppressionfile_BadHeader/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"),
		imported:     false,
		deliveryDate: time.Now(),
	}

	// failed import: file that's over the threshold - should move
	fileTime, _ = time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	metadata2 := &suppressionFileMetadata{
		name:         "T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009",
		timestamp:    fileTime,
		filePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009"),
		imported:     false,
		deliveryDate: fileTime,
	}

	// successful import: should move
	metadata3 := &suppressionFileMetadata{
		name:         "T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420",
		timestamp:    fileTime,
		filePath:     filepath.Join(s.basePath, "suppressionfile_BadFileNames/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420"),
		imported:     true,
		deliveryDate: time.Now(),
	}

	suppresslist = []*suppressionFileMetadata{metadata, metadata2, metadata3}
	err := cleanupSuppression(suppresslist)
	assert.Nil(err)

	files, err := ioutil.ReadDir(conf.GetEnv("PENDING_DELETION_DIR"))
	if err != nil {
		s.FailNow("failed to read directory: %s", conf.GetEnv("PENDING_DELETION_DIR"), err)
	}

	for _, file := range files {
		assert.NotEqual("T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009", file.Name())

		if file.Name() != "T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420" && file.Name() != "T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009" {
			err = fmt.Errorf("unknown file moved %s", file.Name())
			s.FailNow("test files did not correctly cleanup", err)
		}
	}
}
