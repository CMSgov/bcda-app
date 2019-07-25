package public

import (
	"errors"
	"github.com/CMSgov/bcda-app/ssas"
	"strings"
	"time"
)

type MockMFAPlugin struct{}

/*
	VerifyFactorChallenge tests an MFA passcode for validity.  This function should be used for all factor types
	except Push.
*/
func (m *MockMFAPlugin) VerifyFactorChallenge(userIdentifier string, factorType string, passcode string, trackingId string) (bool, error) {
	return false, errors.New("function VerifyFactorTransaction() not yet implemented in MockMFAPlugin")
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
	error@test.com			aborted						none
	(all others)			request_sent				none
*/
func (m *MockMFAPlugin) RequestFactorChallenge(userIdentifier string, factorType string, trackingId string) (factorReturn *FactorReturn, err error) {
	requestEvent := ssas.Event{Op: "RequestOktaFactorChallenge", TrackingID: trackingId}
	ssas.OperationStarted(requestEvent)

	switch strings.ToLower(factorType) {
	case "google totp": fallthrough
	case "okta totp": fallthrough
	case "push": fallthrough
	case "sms": fallthrough
	case "call": fallthrough
	case "email": // noop
	default:
		factorReturn = &FactorReturn{Action: "invalid_request"}
		requestEvent.Help = "invalid factor type: " + factorType
		ssas.OperationFailed(requestEvent)
		return
	}

	switch strings.ToLower(userIdentifier) {
	case "error@test.com":
		requestEvent.Help = "mocking error"
		ssas.OperationFailed(requestEvent)
		return
	case "transaction@test.com":
		transactionId, _ := generateOktaTransactionId()
		transactionExpires := time.Now().Add(time.Minute*5)
		factorReturn = &FactorReturn{Action: "request_sent", Transaction: &Transaction{TransactionID: transactionId, ExpiresAt: transactionExpires}}
	case "success@test.com": fallthrough
	default:
		factorReturn = &FactorReturn{Action: "request_sent"}
	}

	ssas.OperationSucceeded(requestEvent)
	return
}
