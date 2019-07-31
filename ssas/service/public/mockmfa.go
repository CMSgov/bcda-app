package public

import (
	"errors"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/ssas"
)

type MockMFAPlugin struct{}

/*
	VerifyPassword checks a username/password combination for validity, and if successful, returns a value representing
	the state of the account.  It mocks responses according to the following chart:

	userIdentifier			response		error
	--------------			--------		-----
	success@test.com		true			none
	locked_out@test.com		false			none
	mfa_enroll@test.com		false			none
	expired@test.com		false			none
	bad_password@test.com	false			none
	error@test.com			(none)			(non-nil error)
	(all others)			true			none
*/
func (m *MockMFAPlugin) VerifyPassword(userIdentifier string, password string, trackingId string) (passwordReturn *PasswordReturn, err error) {
	r := PasswordReturn{Success: false, Message: ""}
	verifyEvent := ssas.Event{Op: "VerifyOktaPassword", TrackingID: trackingId}
	ssas.OperationStarted(verifyEvent)

	switch strings.ToLower(userIdentifier) {
	case "error@test.com":
		err = errors.New("mocking error")
		verifyEvent.Help = "mocking error"
		passwordReturn = nil
		ssas.OperationFailed(verifyEvent)
		return
	case "locked_out@test.com":
		r.Message = "account locked out"
	case "mfa_enroll@test.com":
		r.Message = "account needs to enroll MFA factor"
	case "expired@test.com":
		r.Message = "password expired"
	case "bad_password@test.com":
		r.Message = "authentication failed"
	case "success@test.com":
		fallthrough
	default:
		r.Success = true
		r.Message = "proceed to MFA verification"
		passwordReturn = &r
		ssas.OperationSucceeded(verifyEvent)
		return
	}

	passwordReturn = &r
	ssas.OperationFailed(verifyEvent)
	return
}

/*
	VerifyFactorChallenge tests an MFA passcode for validity.  This function should be used for all factor types
	except Push.  It mocks responses with valid factor types according to the following chart:

	userIdentifier			response			error
	--------------			--------			-----
	success@test.com 		true    			none
	failure@test.com		false				none
	error@test.com			false				(non-nil error)
	(all others)			false				none
*/
func (m *MockMFAPlugin) VerifyFactorChallenge(userIdentifier string, factorType string, passcode string, trackingId string) (success bool) {
	success = false
	verifyEvent := ssas.Event{Op: "VerifyOktaFactorChallenge", TrackingID: trackingId}
	ssas.OperationStarted(verifyEvent)

	if !ValidFactorType(factorType) {
		verifyEvent.Help = "invalid factor type: " + factorType
		ssas.OperationFailed(verifyEvent)
		return
	}

	switch strings.ToLower(userIdentifier) {
	case "error@test.com":
		verifyEvent.Help = "mocking error"
	case "failure@test.com": // noop
	case "success@test.com":
		fallthrough
	default:
		ssas.OperationSucceeded(verifyEvent)
		success = true
		return
	}

	ssas.OperationFailed(verifyEvent)
	return
}

/*
   VerifyFactorTransaction reports the status of a Push factor's transaction.  Possible non-error states include success,
   rejection, waiting, and timeout.
*/
func (m *MockMFAPlugin) VerifyFactorTransaction(userIdentifier string, factorType string, transactionId string, trackingId string) (string, error) {
	return "", errors.New("function VerifyFactorTransaction() not yet implemented in MockMFAPlugin")
}

/*
	RequestFactorChallenge is to be called from the /authn/request endpoint.  It mocks responses with
	valid factor types according to the following chart:

	userIdentifier			response					error
	--------------			--------					-----
	success@test.com 		request_sent    			none
	transaction@test.com	request_sent, transaction	none
	error@test.com			(none)						(non-nil error)
	(all others)			request_sent				none
*/
func (m *MockMFAPlugin) RequestFactorChallenge(userIdentifier string, factorType string, trackingId string) (factorReturn *FactorReturn, err error) {
	requestEvent := ssas.Event{Op: "RequestOktaFactorChallenge", TrackingID: trackingId}
	ssas.OperationStarted(requestEvent)

	if !ValidFactorType(factorType) {
		factorReturn = &FactorReturn{Action: "invalid_request"}
		requestEvent.Help = "invalid factor type: " + factorType
		ssas.OperationFailed(requestEvent)
		return
	}

	factorReturn = &FactorReturn{Action: "request_sent"}

	switch strings.ToLower(userIdentifier) {
	case "error@test.com":
		err = errors.New("mocking error")
		requestEvent.Help = "mocking error"
		ssas.OperationFailed(requestEvent)
		return
	case "transaction@test.com":
		transactionId, _ := generateOktaTransactionId()
		transactionExpires := time.Now().Add(time.Minute * 5)
		factorReturn = &FactorReturn{Action: "request_sent", Transaction: &Transaction{TransactionID: transactionId, ExpiresAt: transactionExpires}}
	case "success@test.com":
		fallthrough
	default:
	}

	ssas.OperationSucceeded(requestEvent)
	return
}
