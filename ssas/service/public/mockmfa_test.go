package public

import (
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type MockMFATestSuite struct {
	suite.Suite
	o MFAProvider
}

func (s *MockMFATestSuite) TestConfig() {
	s.o = &MockMFAPlugin{}
}

func (s *MockMFATestSuite) TestRequestFactorChallengeSuccess() {
	trackingId := uuid.NewRandom().String()
	userId := "success@test.com"
	factorType := "SMS"

	factorReturn, err := s.o.RequestFactorChallenge(userId, factorType, trackingId)
	assert.Nil(s.T(), err)
	if factorReturn == nil {
		s.FailNow("we expect no errors from the mocked RequestFactorChallenge() for this user ID")
	}
	assert.Equal(s.T(), factorReturn.Action, "request_sent")
	assert.Nil(s.T(), factorReturn.Transaction)
}

func (s *MockMFATestSuite) TestRequestFactorChallengeTransaction() {
	trackingId := uuid.NewRandom().String()
	userId := "transaction@test.com"
	factorType := "SMS"

	factorReturn, err := s.o.RequestFactorChallenge(userId, factorType, trackingId)
	assert.Nil(s.T(), err)
	if factorReturn == nil {
		s.FailNow("we expect no errors from the mocked RequestFactorChallenge() for this user ID")
	}
	assert.Equal(s.T(), factorReturn.Action, "request_sent")
	if factorReturn.Transaction == nil {
		s.FailNow("we expect a Transaction from the mocked RequestFactorChallenge() for this user ID")
	}
	assert.NotNil(s.T(), factorReturn.Transaction.TransactionID)
}

func (s *MockMFATestSuite) TestRequestFactorChallengeError() {
	trackingId := uuid.NewRandom().String()
	userId := "error@test.com"
	factorType := "SMS"

	factorReturn, err := s.o.RequestFactorChallenge(userId, factorType, trackingId)
	if factorReturn == nil {
		s.FailNow("despite the error, we always expect a factorReturn from the mocked RequestFactorChallenge()")
	}
	assert.Equal(s.T(), factorReturn.Action, "request_sent")
	assert.NotNil(s.T(), err)
}

func (s *MockMFATestSuite) TestRequestFactorChallengeRandomUserID() {
	trackingId := uuid.NewRandom().String()
	userId := "asdf@test.com"
	factorType := "SMS"

	factorReturn, err := s.o.RequestFactorChallenge(userId, factorType, trackingId)
	assert.Nil(s.T(), err)
	if factorReturn == nil {
		s.FailNow("we expect no errors from the mocked RequestFactorChallenge() for this user ID")
	}
	assert.Equal(s.T(), factorReturn.Action, "request_sent")
	assert.Nil(s.T(), factorReturn.Transaction)
}

func (s *MockMFATestSuite) TestRequestFactorChallengeBadFactor() {
	trackingId := uuid.NewRandom().String()
	userId := "success@test.com"
	factorType := "Unknown factor type"

	factorReturn, err := s.o.RequestFactorChallenge(userId, factorType, trackingId)
	assert.Nil(s.T(), err)
	if factorReturn == nil {
		s.FailNow("despite the error, we always expect a factorReturn from the mocked RequestFactorChallenge()")
	}
	assert.Equal(s.T(), factorReturn.Action, "invalid_request")
	assert.Nil(s.T(), factorReturn.Transaction)
}

func (s *MockMFATestSuite) TestVerifyFactorChallengeSuccess() {
	trackingId := uuid.NewRandom().String()
	userId := "success@test.com"
	factorType := "SMS"
	passcode := "mock doesn't care what this is"

	success := s.o.VerifyFactorChallenge(userId, factorType, passcode, trackingId)
	assert.True(s.T(), success)
}

func (s *MockMFATestSuite) TestVerifyFactorChallengeFailure() {
	trackingId := uuid.NewRandom().String()
	userId := "failure@test.com"
	factorType := "SMS"
	passcode := "mock doesn't care what this is"

	success := s.o.VerifyFactorChallenge(userId, factorType, passcode, trackingId)
	assert.False(s.T(), success)
}

func (s *MockMFATestSuite) TestVerifyFactorChallengeError() {
	trackingId := uuid.NewRandom().String()
	userId := "error@test.com"
	factorType := "SMS"
	passcode := "mock doesn't care what this is"

	success := s.o.VerifyFactorChallenge(userId, factorType, passcode, trackingId)
	assert.False(s.T(), success)
}

func (s *MockMFATestSuite) TestVerifyFactorChallengeRandomUserID() {
	trackingId := uuid.NewRandom().String()
	userId := "asdf@test.com"
	factorType := "SMS"
	passcode := "mock doesn't care what this is"

	success := s.o.VerifyFactorChallenge(userId, factorType, passcode, trackingId)
	assert.True(s.T(), success)
}

func (s *MockMFATestSuite) TestVerifyFactorChallengeBadFactor() {
	trackingId := uuid.NewRandom().String()
	userId := "success@test.com"
	factorType := "Unknown factor type"
	passcode := "mock doesn't care what this is"

	success := s.o.VerifyFactorChallenge(userId, factorType, passcode, trackingId)
	assert.False(s.T(), success)
}

func TestMockMFATestSuite(t *testing.T) {
	suite.Run(t, new(MockMFATestSuite))
}