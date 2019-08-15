package ssas

import (
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type HashTestSuite struct {
	suite.Suite
}

func (s *HashTestSuite) TestHashComparable() {
	uuidString := uuid.NewRandom().String()
	hash, err := NewHash(uuidString)
	assert.Nil(s.T(), err)
	assert.True(s.T(), hash.IsHashOf(uuidString))
	assert.False(s.T(), hash.IsHashOf(uuid.NewRandom().String()))
}

func (s *HashTestSuite) TestHashUnique() {
	uuidString := uuid.NewRandom().String()
	hash1, _ := NewHash(uuidString)
	hash2, _ := NewHash(uuidString)
	assert.NotEqual(s.T(), hash1.String(), hash2.String())
}

func (s *HashTestSuite) TestHashCompatibility() {
	uuidString := "96c5a0cd-b284-47ac-be6e-f33b14dc4697"
	hash := Hash("YMkApwNDTca4xlM/ROE4ZsiPLrWhjBGbJWue5RghICs=:S/xW9ehijAxxBtsMrDH+R6MYc/l4Sr3Y2SNkPJizy7WW0yaw7FFoAQ1R95WdWnrbPWaM6U0St5U6fp8Bge5pIA==")
	assert.True(s.T(), hash.IsHashOf(uuidString), "Possible change in hashing parameters or algorithm.  Known input/output does not match.  Merging this code will result in invalidating credentials.")
}

func (s *HashTestSuite) TestHashEmpty() {
	hash, err := NewHash("")
	assert.NotNil(s.T(), err)
	assert.False(s.T(), hash.IsHashOf(""))
}

func (s *HashTestSuite) TestHashInvalid() {
	hash := Hash("INVALID_NUMBER_OF_SEGMENTS:d3H4fX/uEk1jOW2gYrFezyuJoSv4ay2x3gH5C25KpWM=:kVqFm1he5S4R1/10oIkVNFot40VB3wTa+DXTp4TrwvyXHkQO7Dxjjo/OqwemiYP8p3UQ8r/HkmTQrSS99UXzaQ==")
	assert.False(s.T(), hash.IsHashOf("96c5a0cd-b284-47ac-be6e-f33b14dc4697"))
}

func TestHashTestSuite(t *testing.T) {
	suite.Run(t, new(HashTestSuite))
}
