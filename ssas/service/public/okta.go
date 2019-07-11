package public

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/okta"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
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
	Result		string	`json:"factorResult"`
}

type OktaClient struct{
	Client 	*http.Client
}

func NewOkta(client *http.Client) *OktaClient {
	if nil == client {
		client = okta.Client()
	}

	return &OktaClient{Client: client}
}

/*
	GetUser searches for Okta users using the provided search string.  Only return results if exactly one active user
	of LOA=3 is found.
 */
func (o *OktaClient) GetUser(searchString string, trackingId string) (oktaId string, err error) {
	userEvent := ssas.Event{Op: "FindOktaUser", TrackingID: trackingId}
	ssas.OperationStarted(userEvent)

	policyUrl := fmt.Sprintf("%s/api/v1/users/?q=%s", okta.OktaBaseUrl, searchString)

	req, err := http.NewRequest("GET", policyUrl, nil)
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

	if resp.StatusCode != 200 {
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
	GetUserFactor looks for the active Okta factor of the specified type enrolled for a given user.

	Valid factor types include:
		"Google TOTP" (Google Authenticator)
		"Okta TOTP"   (Okta Verify app time-based token)
		"Push"        (Okta Verify app push)
		"SMS"
		"Call"
		"Email"
*/
func (o *OktaClient) GetUserFactor(oktaUserId string, factorType string, trackingId string) (factor *Factor, err error) {
	factorEvent := ssas.Event{Op: "FindOktaUserFactors", UserID: oktaUserId, TrackingID: trackingId}
	ssas.OperationStarted(factorEvent)

	policyUrl := fmt.Sprintf("%s/api/v1/users/%s/factors", okta.OktaBaseUrl, oktaUserId)

	req, err := http.NewRequest("GET", policyUrl, nil)
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