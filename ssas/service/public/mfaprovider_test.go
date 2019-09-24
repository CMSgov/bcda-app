package public

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

var origProvider string

type MFAProviderTestSuite struct {
	suite.Suite
}

func (s *MFAProviderTestSuite) SetupTest() {
	origProvider = providerName
}

func (s *MFAProviderTestSuite) TearDownTest() {
	providerName = origProvider
}

func (s *MFAProviderTestSuite) TestSetProvider() {
	SetProvider("")
	assert.Equal(s.T(), Mock, providerName)

	SetProvider("invalid_provider")
	assert.Equal(s.T(), Mock, providerName)

	SetProvider("Mock")
	assert.Equal(s.T(), Mock, providerName)

	SetProvider("mock")
	assert.Equal(s.T(), Mock, providerName)

	SetProvider("Live")
	assert.Equal(s.T(), Live, providerName)

	SetProvider("live")
	assert.Equal(s.T(), Live, providerName)
}

func (s *MFAProviderTestSuite) TestGetProviderName() {
	providerName = "live"
	assert.Equal(s.T(), Live, GetProviderName())

	providerName = "mock"
	assert.Equal(s.T(), Mock, GetProviderName())
}

func (s *MFAProviderTestSuite) TestGetProvider() {
	providerName = "live"
	assert.NotEqual(s.T(), &MockMFAPlugin{}, GetProvider())

	providerName = "mock"
	assert.Equal(s.T(), &MockMFAPlugin{}, GetProvider())

	providerName = "invalid_provider"
	assert.Equal(s.T(), &MockMFAPlugin{}, GetProvider())
}

func (s *MFAProviderTestSuite) TestValidFactorType() {
	assert.Equal(s.T(), true, ValidFactorType("google totp"))
	assert.Equal(s.T(), true, ValidFactorType("Google TOTP"))
	assert.Equal(s.T(), true, ValidFactorType("okta totp"))
	assert.Equal(s.T(), true, ValidFactorType("Okta TOTP"))
	assert.Equal(s.T(), true, ValidFactorType("push"))
	assert.Equal(s.T(), true, ValidFactorType("Push"))
	assert.Equal(s.T(), true, ValidFactorType("sms"))
	assert.Equal(s.T(), true, ValidFactorType("SMS"))
	assert.Equal(s.T(), true, ValidFactorType("call"))
	assert.Equal(s.T(), true, ValidFactorType("Call"))
	assert.Equal(s.T(), true, ValidFactorType("email"))
	assert.Equal(s.T(), true, ValidFactorType("Email"))
	assert.Equal(s.T(), false, ValidFactorType("Any other factor type"))
}


func TestMFAProviderTestSuite(t *testing.T) {
	suite.Run(t, new(MFAProviderTestSuite))
}
