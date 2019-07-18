package public

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/cfg"
	"github.com/CMSgov/bcda-app/ssas/okta"
	"io/ioutil"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type OktaUser struct {
	Id			string	`json:"id"`
	Status		string	`json:"status"`
	Profile	UserProfile	`json:"profile"`
}

type UserProfile struct {
	LOA			string	`json:"LOA,omitempty"`
}

type Factor struct {
	Id			string	`json:"id"`
	Type		string	`json:"factorType"`
	Provider	string	`json:"provider"`
	Status		string	`json:"status"`
}

type FactorRequest struct {
	Result		string		`json:"factorResult"`
	ExpiresAt	time.Time 	`json:"expiresAt,omitempty"`
	Links		OktaLinks	`json:"_links,omitempty"`
}

type OktaLinks struct {
	Cancel		Link	`json:"cancel,omitempty"`
	Poll		Link	`json:"poll,omitempty"`
}

type Allow struct {
	Verbs	[]string	`json:"allow"`
}

type Link struct {
	Href		string	`json:"href"`
	Hints		Allow	`json:"hints"`
}

type OktaMFAPlugin struct{
	Client 	*http.Client
}

var RequestFactorChallengeDuration time.Duration

func init() {
	factorChallengeMilliseconds := cfg.GetEnvInt("SSAS_MFA_CHALLENGE_REQUEST_MILLISECONDS", 1500)
	RequestFactorChallengeDuration = time.Millisecond * time.Duration(factorChallengeMilliseconds)
}

func NewOktaMFA(client *http.Client) *OktaMFAPlugin {
	if nil == client {
		client = okta.Client()
	}

	return &OktaMFAPlugin{Client: client}
}

/*
	VerifyFactorChallenge tests an MFA passcode for validity.  This function should be used for all factor types
	except Push.
 */
func (o *OktaMFAPlugin) VerifyFactorChallenge(userIdentifier string, factorType string, passcode string, trackingId string) (bool, error) {
	return false, errors.New("function VerifyFactorTransaction() not yet implemented in OktaMFAPlugin")
}

/*
   VerifyFactorTransaction reports the status of a Push factor's transaction.  Possible non-error states include success,
   rejection, waiting, and timeout.
 */
func (o *OktaMFAPlugin) VerifyFactorTransaction(userIdentifier string, factorType string, transactionId string, trackingId string) (string, error) {
	return "", errors.New("function VerifyFactorTransaction() not yet implemented in OktaMFAPlugin")
}

/*
	RequestFactorChallenge is to be called from the /authn/request endpoint.  It looks up the Okta user ID and factor ID, calls okta.postFactorChallenge(),
	and filters the information returned to minimize information leakage.

	Valid factor types include:
		"Google TOTP" (Google Authenticator)
		"Okta TOTP"   (Okta Verify app time-based token)
		"Push"        (Okta Verify app push)
		"SMS"
		"Call"
		"Email"
 */
func (o *OktaMFAPlugin) RequestFactorChallenge(userIdentifier string, factorType string, trackingId string) (factorReturn *FactorReturn, err error) {
	startTime := time.Now()
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
		wait(startTime, RequestFactorChallengeDuration)
		return
	}

	oktaUserID, err := o.getUser(userIdentifier, trackingId)
	if err != nil {
		factorReturn = formatFactorReturn(factorType, factorReturn)
		requestEvent.Help = "matching user not found: " + err.Error()
		ssas.OperationFailed(requestEvent)
		wait(startTime, RequestFactorChallengeDuration)
		return
	}

	requestEvent.UserID = oktaUserID
	oktaFactor, err := o.getUserFactor(oktaUserID, factorType, trackingId)
	if err != nil {
		factorReturn = formatFactorReturn(factorType, factorReturn)
		requestEvent.Help = "matching factor not found: " + err.Error()
		ssas.OperationFailed(requestEvent)
		wait(startTime, RequestFactorChallengeDuration)
		return
	}

	factorRequest, err := o.postFactorChallenge(oktaUserID, *oktaFactor, trackingId)
	if err != nil {
		factorReturn = formatFactorReturn(factorType, factorReturn)
		requestEvent.Help = "error requesting challenge for factor: " + err.Error()
		ssas.OperationFailed(requestEvent)
		wait(startTime, RequestFactorChallengeDuration)
		return
	}

	if factorRequest.Links.Poll.Href != "" {
		factorReturn = &FactorReturn{Action: "request_sent"}
		factorReturn.Transaction = &Transaction{}
		factorReturn.Transaction.TransactionID = parsePushTransaction(factorRequest.Links.Poll.Href)
		factorReturn.Transaction.ExpiresAt = factorRequest.ExpiresAt
	}

	factorReturn = formatFactorReturn(factorType, factorReturn)
	requestEvent.Help = fmt.Sprintf("okta.RequestFactorChallenge() execution seconds: %f", time.Now().Sub(startTime).Seconds())
	ssas.OperationSucceeded(requestEvent)
	wait(startTime, RequestFactorChallengeDuration)
	return
}

/*
	formatFactorReturn generates dummy return values if needed
 */
func formatFactorReturn(factorType string, factorReturn *FactorReturn) *FactorReturn {
	if factorReturn == nil || factorReturn.Action == "" {
		factorReturn = &FactorReturn{Action: "request_sent"}
	}

	if strings.ToLower(factorType) == "push" {
		if factorReturn.Transaction == nil || factorReturn.Transaction.TransactionID == "" {
			factorReturn.Transaction = &Transaction{}
			transactionID, err := generateOktaTransactionId()
			if err != nil {
				return &FactorReturn{Action: "aborted"}
			}
			factorReturn.Transaction.TransactionID = transactionID
		}

		if factorReturn.Transaction.ExpiresAt.Before(time.Now()) {
			factorReturn.Transaction.ExpiresAt = time.Now().Add(time.Minute*5)
		}
	} else {
		factorReturn.Transaction = nil
	}
	return factorReturn
}

/*
	wait() provides fixed-time execution for functions that could leak information based on how quickly they return
 */
func wait(startTime time.Time, targetDuration time.Duration) {
	elapsed := time.Now().Sub(startTime)
	time.Sleep(targetDuration - elapsed)
}

/*
	getUser searches for Okta users using the provided search string.  Only return results if exactly one active user
	of LOA=3 is found.
 */
func (o *OktaMFAPlugin) getUser(searchString string, trackingId string) (oktaId string, err error) {
	userEvent := ssas.Event{Op: "FindOktaUser", TrackingID: trackingId}
	ssas.OperationStarted(userEvent)

	userUrl := fmt.Sprintf("%s/api/v1/users/?q=%s", okta.OktaBaseUrl, searchString)

	req, err := http.NewRequest("GET", userUrl, nil)
	if err != nil {
		userEvent.Help = "unable to create request: " + err.Error()
		ssas.OperationFailed(userEvent)
		return "", errors.New(userEvent.Help)
	}

	okta.AddRequestHeaders(req)
	resp, err := o.Client.Do(req)
	if err != nil {
		userEvent.Help = "request error: " + err.Error()
		ssas.OperationFailed(userEvent)
		return "", errors.New(userEvent.Help)
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		userEvent.Help = fmt.Sprintf("unexpected status code %d; unable to read response body", resp.StatusCode)
		ssas.OperationFailed(userEvent)
		return "", errors.New(userEvent.Help)
	}

	if resp.StatusCode >= 400 {
		oktaError, err := okta.ParseOktaError(body)
		if err == nil {
			userEvent.Help = fmt.Sprintf("error received, HTTP response code %d, Okta error %s: %s",
				resp.StatusCode, oktaError.ErrorCode, oktaError.ErrorSummary)
			ssas.OperationFailed(userEvent)
			return "", errors.New(userEvent.Help)
		}
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		userEvent.Help = fmt.Sprintf("unexpected status code %d; response: %s", resp.StatusCode, string(body))
		ssas.OperationFailed(userEvent)
		return "", errors.New(userEvent.Help)
	}

	var users []OktaUser
	if err = json.Unmarshal(body, &users); err != nil {
		userEvent.Help = fmt.Sprintf("unexpected response format; response: %s", string(body))
		ssas.OperationFailed(userEvent)
		return "", errors.New(userEvent.Help)
	}

	var userCountMessage string
	switch {
	case len(users) == 0:
		userCountMessage = "user not found"
	case len(users) > 1:
		userCountMessage = "multiple users found"
	}

	if len(users) != 1 {
		userEvent.Help = fmt.Sprintf("error finding user: %s", userCountMessage)
		ssas.OperationFailed(userEvent)
		return "", errors.New(userEvent.Help)
	}

	user := users[0]
	if user.Status != "ACTIVE" {
		userEvent.Help = "user not active"
		ssas.OperationFailed(userEvent)
		return "", errors.New(userEvent.Help)
	}

	if user.Profile.LOA != "3" {
		userEvent.Help = "user not certified LOA 3"
		ssas.OperationFailed(userEvent)
		return "", errors.New(userEvent.Help)
	}

	return user.Id, nil
}

/*
	getUserFactor looks for the active Okta factor of the specified type enrolled for a given user.

	Valid factor types include:
		"Google TOTP" (Google Authenticator)
		"Okta TOTP"   (Okta Verify app time-based token)
		"Push"        (Okta Verify app push)
		"SMS"
		"Call"
		"Email"
*/
func (o *OktaMFAPlugin) getUserFactor(oktaUserId string, factorType string, trackingId string) (factor *Factor, err error) {
	factorEvent := ssas.Event{Op: "FindOktaUserFactors", UserID: oktaUserId, TrackingID: trackingId}
	ssas.OperationStarted(factorEvent)

	factorUrl := fmt.Sprintf("%s/api/v1/users/%s/factors", okta.OktaBaseUrl, oktaUserId)

	req, err := http.NewRequest("GET", factorUrl, nil)
	if err != nil {
		factorEvent.Help = "unable to create request: " + err.Error()
		ssas.OperationFailed(factorEvent)
		return factor, errors.New(factorEvent.Help)
	}

	okta.AddRequestHeaders(req)
	resp, err := o.Client.Do(req)
	if err != nil {
		factorEvent.Help = "request error: " + err.Error()
		ssas.OperationFailed(factorEvent)
		return factor, errors.New(factorEvent.Help)
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		factorEvent.Help = fmt.Sprintf("unexpected status code %d; unable to read response body", resp.StatusCode)
		ssas.OperationFailed(factorEvent)
		return factor, errors.New(factorEvent.Help)
	}

	if resp.StatusCode >= 400 {
		oktaError, err := okta.ParseOktaError(body)
		if err == nil {
			factorEvent.Help = fmt.Sprintf("error received, HTTP response code %d, Okta error %s: %s",
				resp.StatusCode, oktaError.ErrorCode, oktaError.ErrorSummary)
			ssas.OperationFailed(factorEvent)
			return factor, errors.New(factorEvent.Help)
		}
	}

	if resp.StatusCode != 200 {
		factorEvent.Help = fmt.Sprintf("unexpected status code %d; response: %s", resp.StatusCode, string(body))
		ssas.OperationFailed(factorEvent)
		return factor, errors.New(factorEvent.Help)
	}

	var factors []Factor
	if err = json.Unmarshal(body, &factors); err != nil {
		factorEvent.Help = fmt.Sprintf("unexpected response format; response: %s", string(body))
		ssas.OperationFailed(factorEvent)
		return factor, errors.New(factorEvent.Help)
	}

	for _, f := range factors {
		if f.Status != "ACTIVE" {
			continue
		}

		t := strings.ToLower(factorType)

		switch {
		case t == "google totp" && f.Type == "token:software:totp" && f.Provider == "GOOGLE":
				ssas.OperationSucceeded(factorEvent)
				return &f, nil
		case t == "okta totp" && f.Type == "token:software:totp" && f.Provider == "OKTA":
				ssas.OperationSucceeded(factorEvent)
				return &f, nil
		case t == string(f.Type):
				ssas.OperationSucceeded(factorEvent)
				return &f, nil
		default:
				continue
		}
	}

	factorEvent.Help = fmt.Sprintf("no active factor of requested type %s found", factorType)
	ssas.OperationFailed(factorEvent)
	return factor, errors.New(factorEvent.Help)
}

func (o *OktaMFAPlugin) postFactorChallenge(oktaUserId string, oktaFactor Factor, trackingId string) (factorRequest *FactorRequest, err error) {
	requestEvent := ssas.Event{Op: "PostOktaFactorChallenge", UserID: oktaUserId, TrackingID: trackingId}
	ssas.OperationStarted(requestEvent)

	requestUrl := fmt.Sprintf("%s/api/v1/users/%s/factors/%s/verify", okta.OktaBaseUrl, oktaUserId, oktaFactor.Id)
	req, err := http.NewRequest("POST", requestUrl, nil)
	if err != nil {
		requestEvent.Help = "unable to create request: " + err.Error()
		ssas.OperationFailed(requestEvent)
		return factorRequest, errors.New(requestEvent.Help)
	}

	okta.AddRequestHeaders(req)
	resp, err := o.Client.Do(req)
	if err != nil {
		requestEvent.Help = "request error: " + err.Error()
		ssas.OperationFailed(requestEvent)
		return factorRequest, errors.New(requestEvent.Help)
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		requestEvent.Help = fmt.Sprintf("unexpected status code %d; unable to read response body", resp.StatusCode)
		ssas.OperationFailed(requestEvent)
		return factorRequest, errors.New(requestEvent.Help)
	}

	if resp.StatusCode >= 400 {
		oktaError, err := okta.ParseOktaError(body)
		if err == nil {
			requestEvent.Help = fmt.Sprintf("error received, HTTP response code %d, Okta error %s: %s",
				resp.StatusCode, oktaError.ErrorCode, oktaError.ErrorSummary)
			ssas.OperationFailed(requestEvent)
			return factorRequest, errors.New(requestEvent.Help)
		}
	}

	// HTTP status code 201 is used for push notifications; all others receive 200
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		requestEvent.Help = fmt.Sprintf("unexpected status code %d; response: %s", resp.StatusCode, string(body))
		ssas.OperationFailed(requestEvent)
		return factorRequest, errors.New(requestEvent.Help)
	}

	f := FactorRequest{}
	if err = json.Unmarshal(body, &f); err != nil {
		requestEvent.Help = fmt.Sprintf("unexpected response format; response: %s", string(body))
		ssas.OperationFailed(requestEvent)
		return factorRequest, errors.New(requestEvent.Help)
	}

	ssas.OperationSucceeded(requestEvent)
	return &f, nil
}

/*
	parsePushTransaction returns the Okta transaction ID for a Push factor request
 */
func parsePushTransaction(url string) string {
	re := regexp.MustCompile(`/transactions/(.*)$`)
	matches := re.FindSubmatch([]byte(url))
	if len(matches) > 1 {
		return string(matches[1])
	}

	return ""
}

func generateOktaTransactionId() (string, error) {
	randomPart, err := randomCharacters(22)
	if err != nil {
		return "", errors.New("unable to generate random characters")
	}

	return "v2mst." + randomPart, nil
}

func randomCharacters(length int) (string, error) {
	chars := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")
	randomBytes := make([]byte,length)
	for i := 0; i < length; i++ {
		bign, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", errors.New("unable to generate random number")
		}
		n := bign.Int64()
		randomBytes[i] = chars[n]
	}
	return string(randomBytes), nil
}