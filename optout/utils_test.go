package optout

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type OptOutTestSuite struct {
	suite.Suite
	basePath string
	cleanup  func()
}

func (s *OptOutTestSuite) SetupTest() {
	s.basePath, s.cleanup = testUtils.CopyToTemporaryDirectory(s.T(), "../shared_files/")
}

func (s *OptOutTestSuite) TestParseMetadata() {
	assert := assert.New(s.T())

	// positive
	expTime, _ := time.Parse(time.RFC3339, "2018-11-20T20:13:01Z")
	metadata, err := ParseMetadata("blah/T#EFT.ON.ACO.NGD1800.DPRF.D181120.T2013010")
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D181120.T2013010", metadata.Name)
	assert.Equal(expTime.Format("010203040506"), metadata.Timestamp.Format("010203040506"))
	assert.Nil(err)

	// change the name and timestamp
	expTime, _ = time.Parse(time.RFC3339, "2019-12-20T21:09:42Z")
	metadata, err = ParseMetadata("blah/T#EFT.ON.ACO.NGD1800.DPRF.D191220.T2109420")
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D191220.T2109420", metadata.Name)
	assert.Equal(expTime.Format("010203040506"), metadata.Timestamp.Format("010203040506"))
	assert.Nil(err)
}

func (s *OptOutTestSuite) TestParseMetadata_InvalidFilename() {
	assert := assert.New(s.T())

	// invalid file name
	_, err := ParseMetadata("/path/to/file")
	assert.EqualError(err, "invalid filename for file: /path/to/file")

	_, err = ParseMetadata("/path/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000010")
	assert.EqualError(err, "invalid filename for file: /path/T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000010")

	// invalid date
	_, err = ParseMetadata("/path/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420")
	assert.EqualError(err, "failed to parse date 'D190117.T990942' from file: /path/T#EFT.ON.ACO.NGD1800.DPRF.D190117.T9909420: parsing time \"D190117.T990942\": hour out of range")
}
