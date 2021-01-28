package utils

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
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

	// Iterate through the results, and only encounter "two" once
	// If counter is 1 after the for loop, good... anything else, not good
	var counter = 0

	for _, v := range result {
		if v == "two" {
			counter++
		}
	}

	assert.Equal(s.T(), counter, 1)
	// Ensure "one", "two", "thre" all exists
	assert.Contains(s.T(), result, "one")
	assert.Contains(s.T(), result, "two")
	assert.Contains(s.T(), result, "three")
	// Ensure no additional string was added to the slice
	assert.Len(s.T(), result, 3)
}

func TestCommonTestSuite(t *testing.T) {
	suite.Run(t, new(CommonTestSuite))
}
