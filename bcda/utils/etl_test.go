package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ETLTestSuite struct {
	suite.Suite
}

func TestETLTestSuite(t *testing.T) {
	suite.Run(t, new(ETLTestSuite))
}

func (s *CCLFTestSuite) TestDeleteDirectory() {
	assert := assert.New(s.T())
	dirToDelete := BASE_FILE_PATH + "doomedDirectory"
	testUtils.MakeDirToDelete(s.Suite, dirToDelete)
	defer os.Remove(dirToDelete)

	f, err := os.Open(dirToDelete)
	assert.Nil(err)
	files, err := f.Readdir(-1)
	assert.Nil(err)
	assert.Equal(4, len(files))

	filesDeleted, err := DeleteDirectoryContents(dirToDelete)
	assert.Equal(4, filesDeleted)
	assert.Nil(err)

	f, err = os.Open(dirToDelete)
	assert.Nil(err)
	files, err = f.Readdir(-1)
	assert.Nil(err)
	assert.Equal(0, len(files))

	filesDeleted, err = DeleteDirectoryContents("This/Does/not/Exist")
	assert.Equal(0, filesDeleted)
	assert.EqualError(err, "could not open dir: This/Does/not/Exist: open This/Does/not/Exist: no such file or directory")
}

func makeDirToDelete(s suite.Suite, filePath string) {
	err := os.Mkdir(filePath, os.ModePerm)
	if err != nil {
		s.FailNow("failed to create dir : %s", filePath)
	}
	_, err = os.Create(filepath.Join(filePath, "deleteMe1.txt"))
	if err != nil {
		s.FailNow("failed to create file")
	}
	_, err = os.Create(filepath.Join(filePath, "deleteMe2.txt"))
	if err != nil {
		s.FailNow("failed to create file", filePath)
	}
}
