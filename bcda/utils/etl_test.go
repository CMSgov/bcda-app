package utils

import (
	"io/ioutil"
	"os"
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

func (s *ETLTestSuite) TestDeleteDirectory() {
	assert := assert.New(s.T())
	dirToDelete, err := ioutil.TempDir("", "*")
	assert.NoError(err)

	testUtils.MakeDirToDelete(s.Suite, dirToDelete)
	defer os.RemoveAll(dirToDelete)

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
