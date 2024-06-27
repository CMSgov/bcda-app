package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CommonTestSuite struct {
	suite.Suite
}

// Testing the Dedup() function that removes duplicates in string slice
func (s *CommonTestSuite) TestDedup() {

	// Sample data to be used for testing
	var sampleSlice = []string{"one", "two", "two", "three"}
	// Results should not have 2 "two"s
	var result = Dedup(sampleSlice)

	// Ensure "one", "two", "three" all exists
	assert.Contains(s.T(), result, "one")
	assert.Contains(s.T(), result, "two")
	assert.Contains(s.T(), result, "three")
	// Ensure no additional string was added to the slice
	assert.Len(s.T(), result, 3)
}

func (s *CommonTestSuite) TestSliceCount() {
	isEven := func(n int) bool { return n%2 == 0 }
	assert.Equal(s.T(), 0, SliceCount([]int{}, isEven))
	assert.Equal(s.T(), 0, SliceCount([]int{1, 3}, isEven))
	assert.Equal(s.T(), 1, SliceCount([]int{1, 2, 3}, isEven))
	assert.Equal(s.T(), 2, SliceCount([]int{2, 1, 3, 4}, isEven))
}

func TestCommonTestSuite(t *testing.T) {
	suite.Run(t, new(CommonTestSuite))
}
