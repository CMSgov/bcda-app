package optout

import (
	"fmt"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type OptOutTestSuite struct {
	suite.Suite
}

func TestOptOutTestSuite(t *testing.T) {
	suite.Run(t, new(OptOutTestSuite))
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

func (s *OptOutTestSuite) TestParseMetadata_CorrectEnv() {
	assert := assert.New(s.T())

	originalValue := conf.GetEnv("ENV")
	if err := conf.SetEnv(s.T(), "ENV", "someenv"); err != nil {
		assert.Failf("Error %s env value %s to %s\n", err.Error(), "ENV", "someenv")
	}
	defer func() {
		conf.SetEnv(s.T(), "ENV", originalValue)
	}()

	expTime, _ := time.Parse(time.RFC3339, "2019-12-20T21:09:42Z")
	metadata, err := ParseMetadata("blah/someenv/T#EFT.ON.ACO.NGD1800.DPRF.D191220.T2109420")
	assert.Equal("T#EFT.ON.ACO.NGD1800.DPRF.D191220.T2109420", metadata.Name)
	assert.Equal(expTime.Format("010203040506"), metadata.Timestamp.Format("010203040506"))
	assert.Nil(err)
}

func (s *OptOutTestSuite) TestParseMetadata_DifferentEnv() {
	assert := assert.New(s.T())

	originalValue := conf.GetEnv("ENV")
	if err := conf.SetEnv(s.T(), "ENV", "dev"); err != nil {
		assert.Failf("Error %s env value %s to %s\n", err.Error(), "ENV", "dev")
	}
	defer func() {
		conf.SetEnv(s.T(), "ENV", originalValue)
	}()

	_, err := ParseMetadata("blah/not-someenv/T#EFT.ON.ACO.NGD1800.DPRF.D191220.T2109420")
	assert.EqualError(err, "Skipping file for different environment: blah/not-someenv/T#EFT.ON.ACO.NGD1800.DPRF.D191220.T2109420")
}

func (s *OptOutTestSuite) TestParseMetadata_InvalidData() {
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

func (s *OptOutTestSuite) TestParseRecord_Success() {
	assert := assert.New(s.T())

	// 181120 file
	fileTime, _ := time.Parse(time.RFC3339, "2018-11-20T10:00:00Z")
	line := []byte("5SJ0A00AA001847800005John                          Mitchell                      Doe                                     198203218702 E Fake St.                                        Apt. 63L                                               Region                                                 Las Vegas                               NV423139954M20190618201907011-800TY201907011-800TNT9992WeCare Medical                                                        ")
	metadata := &OptOutFilenameMetadata{
		Timestamp:    fileTime,
		FilePath:     "full-fake-filename",
		Name:         "fake-filename",
		DeliveryDate: time.Now(),
	}

	suppression, err := ParseRecord(metadata, line)
	assert.Nil(err)
	assert.Equal("5SJ0A00AA00", suppression.MBI)
	assert.Equal("1-800", suppression.SourceCode)
}

func (s *OptOutTestSuite) TestParseRecord_InvalidData() {
	assert := assert.New(s.T())
	fp := "testfilepath"

	tests := []struct {
		line   string
		expErr string
	}{
		{
			"1000087481 1847800005John                          Mitchell                      Doe                                     198203218702 E Fake St.                                        Apt. 63L                                               Region                                                 Las Vegas                               NV423139954M20190618201913011-800TY201907011-800TNA9999WeCare Medical                                                        		",
			"failed to parse the effective date '20191301' from file"},
		{"1000087481 1847800005John                          Mitchell                      Doe                                     198203218702 E Fake St.                                        Apt. 63L                                               Region                                                 Las Vegas                               NV423139954M20190618201907011-800TY201913011-800TNA9999WeCare Medical                                                        		",
			"failed to parse the samhsa effective date '20191301' from file"},
		{"1000087481 18e7800005John                          Mitchell                      Doe                                     198203218702 E Fake St.                                        Apt. 63L                                               Region                                                 Las Vegas                               NV423139954M20190618201907011-800TY201907011-800TNA9999WeCare Medical                                                        		",
			"failed to parse beneficiary link key from file"},
	}

	for _, tt := range tests {
		s.T().Run(tt.line, func(t *testing.T) {
			metadata := &OptOutFilenameMetadata{
				Timestamp:    time.Now(),
				FilePath:     fp,
				Name:         tt.line,
				DeliveryDate: time.Now(),
			}
			suppression, err := ParseRecord(metadata, []byte(tt.line))
			assert.Nil(suppression)
			assert.NotNil(err)
			assert.Contains(err.Error(), fmt.Sprintf("%s: %s", tt.expErr, fp))
		})
	}
}
