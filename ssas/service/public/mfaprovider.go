package public

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/okta"
)

const (
	Mock = "mock"
	Live = "live"
)

var providerName = Mock

func init() {
	SetProvider(strings.ToLower(os.Getenv(`SSAS_MFA_PROVIDER`)))
}

func SetProvider(name string) {
	if name != "" {
		switch strings.ToLower(name) {
		case Live:
			providerName = name
		case Mock:
			providerName = name
		default:
			providerEvent := ssas.Event{Op: "SetProvider", Help: fmt.Sprintf(`Unknown providerName %s; using %s`, name, providerName)}
			ssas.ServiceStarted(providerEvent)
		}
	}
	providerEvent := ssas.Event{Op: "SetProvider", Help: fmt.Sprintf(`MFA is made possible by %s`, providerName)}
	ssas.ServiceStarted(providerEvent)
}

func GetProviderName() string {
	return providerName
}

func GetProvider() MFAProvider {
	switch providerName {
	case Live:
		return NewOktaMFA(okta.Client())
	case Mock:
		fallthrough
	default:
		return &MockMFAPlugin{}
	}
}

// FactorReturn defines the return type of RequestFactorChallenge
type FactorReturn struct {
	Action      string       `json:"action"`
	Transaction *Transaction `json:"transaction,omitempty"`
}

// Transaction defines the extra information provided in a response to RequestFactorChallenge for Push factors
type Transaction struct {
	TransactionID string    `json:"transaction_id"`
	ExpiresAt     time.Time `json:"expires_at"`
}

func ValidFactorType(factorType string) bool {
	switch strings.ToLower(factorType) {
	case "google totp":
		fallthrough
	case "okta totp":
		fallthrough
	case "push":
		fallthrough
	case "sms":
		fallthrough
	case "call":
		fallthrough
	case "email":
		return true
	default:
		return false
	}
}

// Provider defines operations performed through an Okta MFA provider.  This indirection allows for a mock provider
// to use during CI/CD integration testing
type MFAProvider interface {
	// RequestFactorChallenge sends an MFA challenge request for the MFA factor type registered to the specified user,
	// if both user and factor exist.  For instance, for the SMS factor type, an SMS message would be sent with a
	// passcode.  Responses for successful and failed attempts should not vary.
	RequestFactorChallenge(userIdentifier string, factorType string, trackingId string) (*FactorReturn, error)

	// VerifyFactorChallenge tests an MFA passcode for validity.  This function should be used for all factor types
	// except Push.
	VerifyFactorChallenge(userIdentifier string, factorType string, passcode string, trackingId string) bool

	// VerifyFactorTransaction reports the status of a Push factor's transaction.  Possible non-error states include success,
	// rejection, waiting, and timeout.
	VerifyFactorTransaction(userIdentifier string, factorType string, transactionId string, trackingId string) (string, error)
}
