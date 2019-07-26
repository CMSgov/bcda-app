// +build okta

// To enable this test suite, either:

// - Run from your IDE after setting the env vars below
//   OR
// - Run "DATABASE_URL=postgresql://postgres:toor@127.0.0.1:5432/bcda?sslmode=disable go test -tags=okta -v" from the ssas/service/public directory

package public

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/ssas/okta"
)

type OktaLiveTestSuite struct {
	suite.Suite
	oc *OktaMFAPlugin
	email string
	userId string
	password string
	smsFactorId string
}

func (s *OktaLiveTestSuite) SetupSuite() {
	s.email = os.Getenv("OKTA_MFA_EMAIL")
	s.userId = os.Getenv("OKTA_MFA_USER_ID")
	s.password = os.Getenv("OKTA_MFA_USER_PASSWORD")
	s.smsFactorId = os.Getenv("OKTA_MFA_SMS_FACTOR_ID")

	if s.email == "" || s.userId == "" || s.password == "" || s.smsFactorId == "" {
		s.FailNow(fmt.Sprintf("Cannot run live Okta tests without env vars set: OKTA_MFA_EMAIL=%s; OKTA_MFA_USER_ID=%s; OKTA_MFA_USER_PASSWORD=%s; OKTA_MFA_SMS_FACTOR_ID=%s",
			s.email, s.userId, s.password, s.smsFactorId))
	}
}

func (s *OktaLiveTestSuite) SetupTest() {
	s.oc = NewOktaMFA(okta.Client())
}

func (s *OktaLiveTestSuite) TestPostPasswordSuccess() {
	trackingId := uuid.NewRandom().String()

	passwordRequest, err := s.oc.postPassword(s.email, s.password, trackingId)
	if err != nil || passwordRequest == nil {
		s.FailNow("password result not parsed: " + err.Error())
	}
	assert.Equal(s.T(), "MFA_REQUIRED", passwordRequest.Status)
}

func (s *OktaLiveTestSuite) TestPostPasswordFailure() {
	trackingId := uuid.NewRandom().String()

	passwordRequest, err := s.oc.postPassword(s.userId, "bad_password", trackingId)
	if err != nil || passwordRequest == nil {
		s.FailNow("password result not parsed: " + err.Error())
	}
	assert.Equal(s.T(), "AUTHENTICATION_FAILED", passwordRequest.Status)
}

func (s *OktaLiveTestSuite) TestPostPasswordBadUserId() {
	trackingId := uuid.NewRandom().String()

	passwordRequest, err := s.oc.postPassword("bad_user_id", s.password, trackingId)
	if err != nil || passwordRequest == nil {
		s.FailNow("password result not parsed: " + err.Error())
	}
	assert.Equal(s.T(), "AUTHENTICATION_FAILED", passwordRequest.Status)
}

func (s *OktaLiveTestSuite) TestPostFactorChallengeSuccess() {
	trackingId := uuid.NewRandom().String()
	factor := Factor{Id: s.smsFactorId, Type: "sms"}

	factorVerification, err := s.oc.postFactorChallenge(s.userId, factor, trackingId)
	if err != nil || factorVerification == nil {
		s.FailNow("factor result not parsed: " + err.Error())
	}
	assert.Equal(s.T(), "CHALLENGE", factorVerification.Result)

	// A second SMS request within 30 seconds will fail.  This second test case must
	// follow in the same unit test as the successful test case.
	factorVerification, err = s.oc.postFactorChallenge(s.userId, factor, trackingId)
	if err == nil || factorVerification != nil {
		s.FailNow("second request should not be successful")
	}
}

func (s *OktaLiveTestSuite) TestPostFactorChallengeInvalidFactor() {
	trackingId := uuid.NewRandom().String()
	factor := Factor{Id: "abcdefg1234567", Type: "sms"}

	factorVerification, err := s.oc.postFactorChallenge(s.userId, factor, trackingId)
	if err == nil || factorVerification != nil {
		s.FailNow("invalid factor should not be successful")
	}
}

func (s *OktaLiveTestSuite) TestPostFactorChallengeInvalidUser() {
	trackingId := uuid.NewRandom().String()
	userId := "abcdefg1234567"
	factor := Factor{Id: s.smsFactorId, Type: "sms"}

	factorVerification, err := s.oc.postFactorChallenge(userId, factor, trackingId)
	if err == nil || factorVerification != nil {
		s.FailNow("invalid factor should not be successful")
	}
}

func (s *OktaLiveTestSuite) TestPinnedKeyNotMatched() {
	originalOktaCACertFingerprint := okta.OktaCACertFingerprint
	okta.OktaCACertFingerprint, _ = hex.DecodeString("00112233aabbcc")
	s.oc = NewOktaMFA(okta.Client())

	trackingId := uuid.NewRandom().String()
	factor := Factor{Id: s.smsFactorId, Type: "sms"}

	factorVerification, err := s.oc.postFactorChallenge(s.userId, factor, trackingId)
	if err == nil || factorVerification != nil {
		s.FailNow("certificate signed by unexpected CA should abort TLS handshake")
	}
	okta.OktaCACertFingerprint = originalOktaCACertFingerprint
}

func (s *OktaLiveTestSuite) TestGetUserSuccess() {
	trackingId := uuid.NewRandom().String()

	foundUserId, err := s.oc.getUser(s.email, trackingId)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), s.userId, foundUserId)
}

func (s *OktaLiveTestSuite) TestGetUserBadAuth() {
	originalOktaAuthString := okta.OktaAuthString
	okta.OktaAuthString = "SSWS 00112233aabbcc"

	trackingId := uuid.NewRandom().String()

	foundUserId, err := s.oc.getUser(s.email, trackingId)
	if err == nil || foundUserId != "" {
		s.FailNow("bad Okta token should not be successful")
	}
	assert.Contains(s.T(), err.Error(), "error received")
	okta.OktaAuthString = originalOktaAuthString
}

func (s *OktaLiveTestSuite) TestGetUserTooManyFound() {
	trackingId := uuid.NewRandom().String()
	searchString := "bcda"

	foundUserId, err := s.oc.getUser(searchString, trackingId)
	if err == nil || foundUserId != "" {
		s.FailNow("user search string with multiple matches should not be successful")
	}
	assert.Contains(s.T(), err.Error(), "multiple users")
}

func (s *OktaLiveTestSuite) TestGetUserNoneFound() {
	trackingId := uuid.NewRandom().String()
	searchString := "a1b2c3d4"

	foundUserId, err := s.oc.getUser(searchString, trackingId)
	if err == nil || foundUserId != "" {
		s.FailNow("user search string with no matches should not be successful")
	}
	assert.Contains(s.T(), err.Error(), "not found")
}

func (s *OktaLiveTestSuite) TestGetUserFactorSuccess() {
	trackingId := uuid.NewRandom().String()
	factorType := "SMS"

	factor, err := s.oc.getUserFactor(s.userId, factorType, trackingId)
	assert.Nil(s.T(), err)
	if factor == nil {
		s.FailNow("getUserFactor() should successfully return a factor")
	}
	assert.Equal(s.T(), s.smsFactorId, factor.Id)
	assert.Equal(s.T(), "sms", factor.Type)
}

func (s *OktaLiveTestSuite) TestGetUserBadUser() {
	trackingId := uuid.NewRandom().String()
	userId := "abc123"
	factorType := "SMS"

	factor, err := s.oc.getUserFactor(userId, factorType, trackingId)
	if err == nil || factor != nil {
		s.FailNow("getUserFactor() should fail with a bad user ID")
	}
	assert.Contains(s.T(), err.Error(), "error received")
}

func (s *OktaLiveTestSuite) TestGetUserFactorNotFound() {
	trackingId := uuid.NewRandom().String()
	factorType := "Push"

	factor, err := s.oc.getUserFactor(s.userId, factorType, trackingId)
	if err == nil || factor != nil {
		s.FailNow("getUserFactor() should fail with factor type to registered to specified user")
	}
	assert.Contains(s.T(), err.Error(), "active factor")
}

func TestOktaLiveTestSuite(t *testing.T) {
	suite.Run(t, new(OktaLiveTestSuite))
}
