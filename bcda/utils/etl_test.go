package utils

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"os"
	"path/filepath"
	"testing"
)

type ETLTestSuite struct {
	suite.Suite
}

func TestETLTestSuite(t *testing.T) {
	suite.Run(t, new(ETLTestSuite))
}

func (s *ETLTestSuite) TestDeleteDirectory() {
	assert := assert.New(s.T())
	dirToDelete := "../../shared_files/doomedDirectory"
	makeDirToDelete(s.Suite, dirToDelete)
	defer os.Remove(dirToDelete)

	f, err := os.Open(dirToDelete)
	assert.Nil(err)
	files, err := f.Readdir(-1)
	assert.Nil(err)
	assert.Equal(2, len(files))

	filesDeleted, err := DeleteDirectoryContents(dirToDelete)
	assert.Equal(2, filesDeleted)
	assert.Nil(err)

	f, err = os.Open(dirToDelete)
	assert.Nil(err)
	files, err = f.Readdir(-1)
	assert.Nil(err)
	assert.Equal(0, len(files))

	filesDeleted, err = DeleteDirectoryContents("This/Does/not/Exist")
	assert.Equal(0, filesDeleted)
	assert.NotNil(err)
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
