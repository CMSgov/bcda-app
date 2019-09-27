package cfg

import (
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/suite"
"os"
"testing"
)

type EnvvTestSuite struct {
suite.Suite
}

var origTestKey string

func (s *EnvvTestSuite) SetupTest() {
origTestKey = os.Getenv("TEST_KEY")
}

func (s *EnvvTestSuite) TearDownTest() {
os.Setenv("TEST_KEY", origTestKey)
}

func (s *EnvvTestSuite) TestFromEnvUnset() {
os.Unsetenv("TEST_KEY")
val := FromEnv("TEST_KEY", "test_val")
assert.Equal(s.T(), "test_val", val)
}

func (s *EnvvTestSuite) TestFromEnvSet() {
os.Setenv("TEST_KEY", "set_val")
val := FromEnv("TEST_KEY", "test_val")
assert.Equal(s.T(), "set_val", val)
}

func (s *EnvvTestSuite) TestGetEnvIntUnset() {
os.Unsetenv("TEST_KEY")
i := GetEnvInt("TEST_KEY", 33)
assert.Equal(s.T(), 33, i)
}

func (s *EnvvTestSuite) TestGetEnvIntSet() {
os.Setenv("TEST_KEY", "55")
i := GetEnvInt("TEST_KEY", 33)
assert.Equal(s.T(), 55, i)
}

func (s *EnvvTestSuite) TestGetEnvIntFloat() {
os.Setenv("TEST_KEY", "5.5")
i := GetEnvInt("TEST_KEY", 33)
assert.Equal(s.T(), 33, i)
}

func (s *EnvvTestSuite) TestGetEnvIntString() {
os.Setenv("TEST_KEY", "Not an int value")
i := GetEnvInt("TEST_KEY", 33)
assert.Equal(s.T(), 33, i)
}

func TestEnvvTestSuite(t *testing.T) {
suite.Run(t, new(EnvvTestSuite))
}