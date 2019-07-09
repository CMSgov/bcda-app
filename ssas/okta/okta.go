package okta

import (
	"fmt"
	"os"
)

var oktaBaseUrl string
var oktaAuthString string
var oktaServerID string

func init() {
	_ = config()
}

// separate from init for testing
func config() error {
	oktaBaseUrl = os.Getenv("OKTA_CLIENT_ORGURL")
	oktaServerID = os.Getenv("OKTA_OAUTH_SERVER_ID")
	oktaToken := os.Getenv("OKTA_CLIENT_TOKEN")

	at := oktaToken
	if at != "" {
		at = "[Redacted]"
	}

	if oktaBaseUrl == "" || oktaServerID == "" || oktaToken == "" {
		return fmt.Errorf(fmt.Sprintf("missing env vars: OKTA_CLIENT_ORGURL=%s, OKTA_OAUTH_SERVER_ID=%s, OKTA_CLIENT_TOKEN=%s", oktaBaseUrl, oktaServerID, at))
	}

	oktaAuthString = fmt.Sprintf("SSWS %s", oktaToken)

	return nil
}
