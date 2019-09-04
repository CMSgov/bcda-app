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
| SSAS_MFA_PROVIDER    | No       | Switches between mock Okta MFA calls and live calls.  Defaults to "Mock".
| SSAS_MFA_CHALLENGE_REQUEST_MILLISECONDS | No | Minimum execution time for RequestFactorChallenge().  If not present, defaults to 1500.  In production, this should always be set longer than the longest expected execution time.  (Actual execution time is logged.)|
| SSAS_MFA_TOKEN_TIMEOUT_MINUTES | No | Token lifetime for self-registration (MFA tokens and Registration tokens).  Defaults to 60 (minutes). |
| OKTA_CLIENT_ORGURL   | Yes      | Sets the URL for contacting Okta (will vary between production/non-production environments). |
| OKTA_CLIENT_TOKEN    | Yes      | A token providing limited admin-level API rights to Okta. |
| OKTA_CA_CERT_FINGERPRINT | Yes  | SHA1 fingerprint for the CA certificate signing the Okta TLS cert.  If the fingerprint does not match the CA certificate presented when we visit Okta, the HTTPS connection is terminated |
| OKTA_MFA_EMAIL       | No       | The email address (Okta account identifier) for the account to test in the Okta sandbox. Required only if running the live Okta MFA tests. |
| OKTA_MFA_USER_ID     | No       | The user ID for the account to test in the Okta sandbox. Required only if running the live Okta MFA tests. |
| OKTA_MFA_USER_PASSWORD| No      | The password for the account to test in the Okta sandbox. Required only if running the live Okta MFA tests. |
| OKTA_MFA_SMS_FACTOR_ID | No     | The SMS MFA factor ID enrolled for the account to test in the Okta sandbox. Required only if running the live Okta MFA tests. |
| SSAS_READ_TIMEOUT    | No       | Sets the read timeout on server requests |
| SSAS_WRITE_TIMEOUT   | No       | Sets the write timeout on server responses |
| SSAS_IDLE_TIMEOUT    | No       | Sets the idle timeout on |
| SSAS_ADMIN_SIGNING_KEY_PATH  | Yes | Provides the location of the admin server signing key |
| SSAS_PUBLIC_SIGNING_KEY_PATH | Yes | Provides the location of the public server signing key |
| SSAS_TOKEN_BLACKLIST_CACHE_CLEANUP_MINUTES | No | Tunes the frequency that expired entries are cleared from the token blacklist cache.  Defaults to 15 minutes. |
| SSAS_TOKEN_BLACKLIST_CACHE_TIMEOUT_MINUTES | No | Sets the lifetime of token blacklist cache entries.  Defaults to 24 hours.
| SSAS_TOKEN_BLACKLIST_CACHE_REFRESH_MINUTES | No | Configures the number of minutes between times the token blacklist cache is refreshed from the database. |

# Build

Build all the code and containers with `make docker-bootstrap`. Alternatively, `docker-compose up ssas` will build and run the SSAS by itself. Note that SSAS needs the db container to be running as well.

## Bootstrapping CLI

SSAS currently has a simple CLI intended to make bootstrapping tasks and manual testing easier to accomplish. The CLI will only run one command at a time; commands do not chain.

The sequence of commands needed to bootstrap the SSAS into a new environment is as follows:

1. migrate, which will build or update the tables
1. add-fixture-data, which adds the admin group and seeds minimal data for smoke Testing
1. new-admin-system, which adds an admin system and returns its client_id
1. reset-secret, which replaces the secret associated with a client_id and returns that new secret
1. start, which starts the servers and the token blacklist cache

You will need the admin client_id and secret to use the service's admin endpoints.

Note that to initialize our docker container, we use migrate-and-start, which combines the first three of the steps above with some conditional logic to make sure we're running in a development environment. This command should most likely not be used elsewhere.

# Test

The SSAS can be tested by running `make test-ssas` or `make unit-test-ssas`. You can also use the repo-wide commands `make test` and `make unit-test`, which will run tests against the entire repo, including the SSAS code.  Some tests are designed to be only run as needed, and are excluded from `make` by a build tag.  To include
one of these test suites, follow the instructions at the top of the test file.

# Integration Testing

To run postman tests locally:

Build and startup the required containers. Building with docker-compose up first will significantly improve the performance of the following steps.

```
docker-compose up
docker-compose stop
docker-compose up -d db
docker-compose up ssas
```

Seed the database with a minimal group:

```
docker run --rm --network bcda-app_default -it postgres psql -h bcda-app_db_1 -U postgres bcda
	insert into groups(group_id) values ('T0000');
```

point your browser at one of the following ports, or use the postman test collection in tests.

- public server: 3003
- admin server: 3004
- forwarding server: 3005


# Goland IDE

To run a test suite inside of Goland IDE, edit its configuration from the `Run` menu and add values for all necessary
environmental variables.  It is also possible to run individual tests, but that may require configurations for each test.

# Docker Fun

```
docker run --rm --network bcda-app_default -it postgres pg_dump -s -h bcda-app_db_1 -U postgres bcda > schema.sql
```
```
docker-compose run --rm ssas sh -c 'tmp/ssas-service --reset-secret --client-id=[client_id]'
```
```
docker-compose run --rm ssas sh -c 'tmp/ssas-service --new-admin-system --system-name=[entity name]'
```
