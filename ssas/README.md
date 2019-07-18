# System-to-System Authentication Service (SSAS)

The SSAS can be run as a standalone web service or embedded as a library. 

# Code Organization

The service package contains a standalone http service that encapsulates the authorization library.

Imports always go up the directory tree from leaves; that is, parents do not import from their children. Children may import from their siblings. In order to maintain the service and library distinction, the ssas package and packages in plugins must not import from packages in the service directory. 

The outline below shows physical the directory structure of the code, with package names highlighted. The service and plugins directories are used for organization, but are not packages themselves. The main package is located in the service directory. Currently, the go files listed are the intended locations of code that will be migrated from bcda auth when an in-progress refactor is completed. The go files currently in place are placeholders that served to set up the code tree and illustrate the intended relationship between the service (main.go) and the ssas. 

Not shown: _test files for every .go file are assumed to be present parallel to their test subject.

- **ssas**
    - **cfg**
      - envv.go
      - _configuration management; nothing in cfg should import from ssas packages_
    - _service_
      - **admin**
        - api.go
        - middleware.go
        - router.go
      - **public**
        - api.go
        - middleware.go
        - router.go
      - **server**
      - main.go
    - _plugins_
      - **alpha**
        - alpha.go
        - backend.go
      - **okta**
        - mokta.go
        - okta.go
        - oktaclient.go
        - oktajwk.go
    - provider.go
    - logger.go
    - credentials.go
      - _data model and functions related to credentials_
    - groups.go
      - _data model and functions related to groups_
    - systems.go
      - _data model and functions related to systems_
    - tokentools.go
      - _functions related to tokens; should perhaps be renamed tokens_
      
# Configuration

Required values must be present in the docker-compose.*.yml files.

| Key                  | Required | Purpose | 
| -------------------- |:--------:| ------- |
| SSAS_HASH_ITERATIONS | Yes      | Controls how many iterations our secure hashing mechanism performs. Service will panic if this key does not have a value. |
| SSAS_HASH_KEY_LENGTH | Yes      | Controls the key length used by our secure hashing mechanism. Service will panic if this key does not have a value. |
| SSAS_HASH_SALT_SIZE  | Yes      | Controls salt size used by our secure hashing mechanism performs. Service will panic if this key does not have a value. |
| SSAS_MFA_CHALLENGE_REQUEST_MILLISECONDS | No | Minimum execution time for RequestFactorChallenge().  If not present, defaults to 1500.  In production, this should always be set longer than the longest expected execution time.  (Actual execution time is logged.)|
| OKTA_CLIENT_ORGURL   | Yes      | Sets the URL for contacting Okta (will vary between production/non-production environments). |
| OKTA_CLIENT_TOKEN    | Yes      | A token providing limited admin-level API rights to Okta. |
| OKTA_CA_CERT_FINGERPRINT | Yes  | SHA1 fingerprint for the CA certificate signing the Okta TLS cert.  If the fingerprint does not match the CA certificate presented when we visit Okta, the HTTPS connection is terminated |
| OKTA_MFA_EMAIL       | No       | The email address (Okta account identifier) for the account to test in the Okta sandbox. Required only if running the live Okta MFA tests. |
| OKTA_MFA_USER_ID     | No       | The user ID for the account to test in the Okta sandbox. Required only if running the live Okta MFA tests. |
| OKTA_MFA_SMS_FACTOR_ID | No     | The SMS MFA factor ID enrolled for the account to test in the Okta sandbox. Required only if running the live Okta MFA tests. |

# Build

Build the code with `make docker-bootstrap`. Alternatively, `docker-compose up ssas` will build and run the SSAS by itself.

# Test

The SSAS can be tested by running `make test` or `make unit-test`.  Some tests are designed to be only run as needed, and
are excluded from `make unit-test` by a build tag.  To include one of these test suites, follow the instructions at the
top of the test file.

# Goland IDE

To run a test suite inside of Goland IDE, edit its configuration from the `Run` menu and add values for all necessary
environmental variables.  It is also possible to run individual tests, but that may require configurations for each test.
