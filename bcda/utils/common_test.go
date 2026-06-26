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

func (s *CommonTestSuite) TestCountUniq() {
	firstLetter := func(s string) string { return string(s[0]) }
	assert.Equal(s.T(), 0, CountUniq([]string{}, firstLetter))
	assert.Equal(s.T(), 1, CountUniq([]string{"abc", "ab"}, firstLetter))
	assert.Equal(s.T(), 2, CountUniq([]string{"abc", "bcd", "ab"}, firstLetter))
}

func (s *CommonTestSuite) TestMinutesToSeconds() {
	tests := []struct {
		name       string
		minutesStr string
		expected   string
	}{
		{"valid positive minutes", "5", "300"},
		{"valid positive minutes larger", "10", "600"},
		{"invalid format non-numeric", "invalid", "300"},
		{"empty string", "", "300"},
		{"zero minutes", "0", "300"},
		{"negative minutes", "-5", "300"},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			actual := MinutesToSeconds(tt.minutesStr)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestCommonTestSuite(t *testing.T) {
	suite.Run(t, new(CommonTestSuite))
}
