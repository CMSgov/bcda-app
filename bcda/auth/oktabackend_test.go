// +build okta

// To enable this test suite:
// 1. Put an appropriate url into env var OKTA_OAUTH_SERVER_METADATA
// 3. Run "go test -tags=okta" from the bcda/auth directory

package auth

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOB_GetPublicKeys(t *testing.T) {
	publicKeys, err := getPublicKeys()
	assert.Nil(t, err)
	assert.Condition(t, func() bool {return 1 <= len(publicKeys) && len(publicKeys) <= 3})
}

func TestOB_Config(t *testing.T) {
	originalOktaBaseUrl := os.Getenv("OKTA_CLIENT_ORGURL")
	originalOktaServerID := os.Getenv("OKTA_OAUTH_SERVER_ID")
	originalOktaToken := os.Getenv("OKTA_CLIENT_TOKEN")

	os.Unsetenv("OKTA_CLIENT_ORGURL")
	os.Unsetenv("OKTA_OAUTH_SERVER_ID")
	os.Unsetenv("OKTA_CLIENT_TOKEN")

	err := config()
	require.NotNil(t, err)
	assert.Regexp(t, regexp.MustCompile("(OKTA_[A-Z_]*=, ){2}(OKTA_CLIENT_TOKEN=)"), err)

	os.Setenv("OKTA_CLIENT_TOKEN", originalOktaToken)

	err = config()
	assert.NotNil(t, err)
	assert.Regexp(t, regexp.MustCompile("(OKTA_[A-Z_]*=, ){2}(OKTA_CLIENT_TOKEN=[Redacted])"), err)

	os.Setenv("OKTA_CLIENT_ORGURL", originalOktaBaseUrl)
	os.Setenv("OKTA_OAUTH_SERVER_ID", originalOktaServerID)
	os.Setenv("OKTA_CLIENT_TOKEN", originalOktaToken)

	err = config()
	assert.Nil(t, err)
}

func TestOB_ParseKeys(t *testing.T) {
	// sample okta public server signing keys from public documentation site
	// https://developer.okta.com/docs/api/resources/oidc#well-knownoauth-authorization-server
	const sampleJWKS = `
{
    "keys": [
 	    {
		    "alg": "RS256",
		    "e": "AQAB",
		    "n": "iKqiD4cr7FZKm6f05K4r-GQOvjRqjOeFmOho9V7SAXYwCyJluaGBLVvDWO1XlduPLOrsG_Wgs67SOG5qeLPR8T1zDK4bfJAo1TvbwYeTwVSfd_0mzRq8WaVc_2JtEK7J-4Z0MdVm_dJmcMHVfDziCRohSZthN__WM2NwGnbewWnla0wpEsU3QMZ05_OxvbBdQZaDUsNSx46is29eCdYwhkAfFd_cFRq3DixLEYUsRwmOqwABwwDjBTNvgZOomrtD8BRFWSTlwsbrNZtJMYU33wuLO9ynFkZnY6qRKVHr3YToIrqNBXw0RWCheTouQ-snfAB6wcE2WDN3N5z760ejqQ",
		    "kid": "U5R8cHbGw445Qbq8zVO1PcCpXL8yG6IcovVa3laCoxM",
		    "kty": "RSA",
		    "use": "sig"
	    },
		{
			"alg": "RS256",
			"e": "AQAB",
			"n": "l1hZ_g2sgBE3oHvu34T-5XP18FYJWgtul_nRNg-5xra5ySkaXEOJUDRERUG0HrR42uqf9jYrUTwg9fp-SqqNIdHRaN8EwRSDRsKAwK3HIJ2NJfgmrrO2ABkeyUq6rzHxAumiKv1iLFpSawSIiTEBJERtUCDcjbbqyHVFuivIFgH8L37-XDIDb0XG-R8DOoOHLJPTpsgH-rJeM5w96VIRZInsGC5OGWkFdtgk6OkbvVd7_TXcxLCpWeg1vlbmX-0TmG5yjSj7ek05txcpxIqYu-7FIGT0KKvXge_BOSEUlJpBhLKU28OtsOnmc3NLIGXB-GeDiUZiBYQdPR-myB4ZoQ",
			"kid": "Y3vBOdYT-l-I0j-gRQ26XjutSX00TeWiSguuDhW3ngo",
			"kty": "RSA",
			"use": "sig"
		},
		{
			"alg": "RS256",
			"e": "AQAB",
			"n": "lC4ehVB6W0OCtNPnz8udYH9Ao83B6EKnHA5eTcMOap_lQZ-nKtS1lZwBj4wXRVc1XmS0d2OQFA1VMQ-dHLDE3CiGfsGqWbaiZFdW7UGLO1nAwfDdH6xp3xwpKOMewDXbAHJlXdYYAe2ap-CE9c5WLTUBU6JROuWcorHCNJisj1aExyiY5t3JQQVGpBz2oUIHo7NRzQoKimvpdMvMzcYnTlk1dhlG11b1GTkBclprm1BmOP7Ltjd7aEumOJWS67nKcAZzl48Zyg5KtV11V9F9dkGt25qHauqFKL7w3wu-DYhT0hmyFcwn-tXS6e6HQbfHhR_MQxysLtDGOk2ViWv8AQ",
			"kid": "h5Sr3LXcpQiQlAUVPdhrdLFoIvkhRTAVs_h39bQnxlU",
			"kty": "RSA",
			"use": "sig"
		}
	]
}`

	_, err := parseKeys([]byte(sampleJWKS))
	if err != nil {
		assert.FailNow(t, "Failed parsing keys because %s", err)
	}
}

func TestOB_AddClientApplication(t *testing.T) {
	ob := NewOB()
	randomID, _ := someRandomBytes(9)
	clientID, secret, err := ob.AddClientApplication(fmt.Sprintf("BCDA %x", randomID))
	fmt.Printf("\n%s, %s, %s\n", clientID, secret, err)
	assert.Nil(t, err)
	assert.NotEmpty(t, clientID)
	assert.NotEmpty(t, secret)
}
