# System-to-System Authentication Service (SSAS)

The SSAS can be run as a standalone web service or embedded as a library.

# Code Organization

The outline below shows the physical directory structure of the code, with package names highlighted. The service package contains a standalone http service that presents the authorization library via two http servers, one for admin tasks and one for authorization tasks.

Imports always go up the directory tree from leaves; that is, parents do not import from their children. Children may import from their siblings. In short, the `ssas` and `okta` packages must not import from packages in the service directory.

- **ssas**
    - **cfg**
      - _configuration management; cfg should not import from ssas packages_
    - **okta**
    - **service**
      - **admin**
        - _contains the REST API for managing the service implementation_
      - **main**
        - _cli for running servers and some admin tasks_
      - **public**
        - _contains the rest API for authorization services_

# Configuration

Required values must be present in the docker-compose.*.yml files. Some values are primarily for the use of the ACO API, and are only used by SSAS for testing purposes. Some values are only used by the ACO API; they are listed for reference.

Very long keys have been split across two rows for formatting purposes.

Some variables below have a note indicating their name should be changed. These changes serve to make the names consistent with established naming patterns and/or to clarify their purpose. They should be made after we complete the initial deployments to AWS envs so that we don't have to change all of our existing deployment checklists in a short timeframe.

|  Key                 | Required | SSAS | BCDA | Purpose |
| -------------------- |:--------:|:----:|:---:| ------- |
| BCDA_AUTH_PROVIDER   | Yes      |   | X | Tells ACO API which auth provider to use |
| BCDA_CA_FILE         | Yes      |   | X | Tells ACO API the certificate file with which to validate its TLS connection to SSAS. When setting vars for AWS envs, you must include a var for the key material  |
| BCDA_SSAS_CLIENT_ID  | Yes      |   | X | Tells ACO API the client_id to use with the SSAS REST API. |
| BCDA_SSAS_SECRET     | Yes      |   | X | Tells ACO API the secret to use with the SSAS REST API. |
| SSAS_USE_TLS         | Yes      |   | X | Should be renamed to BCDA_SSAS_USE_TLS |
| SSAS_URL             | Yes      |   | X | The url of the SSAS admin server. Should be renamed to BCDA_SSAS_URL |
| SSAS_PUBLIC_URL      | Yes      |   | X | The url of the SSAS public server (auth endpoints). Should be renamed to BCDA_SSAS_URL_PUBLIC |
| DATABASE_URL         | Yes      | X |   | Provides the database url |
| DEBUG                | Depends  | X |   | Flag to indicate that the system is running in a development environments. Generally not used outside of docker. | | |
| HTTP_ONLY            | Depends  | X |   | Flag to operation of the system. By default, the servers will use https. When HTTP_ONLY is present **and** set to true, they will use http. Generally not used outside of docker. |
| OKTA_CLIENT_ORGURL   | Yes      | X |   | Sets the URL for contacting Okta (will vary between production/non-production environments). |
| OKTA_CLIENT_TOKEN    | Yes      | X |   | A token providing limited admin-level API rights to Okta. |
| OKTA_CA_CERT_FINGERPRINT | Yes  | X |   | SHA1 fingerprint for the CA certificate signing the Okta TLS cert.  If the fingerprint does not match the CA certificate presented when we visit Okta, the HTTPS connection is terminated |
| OKTA_MFA_EMAIL       | No       | X |   | The email address (Okta account identifier) for the account to test in the Okta sandbox. Required only if running the live Okta MFA tests. |
| OKTA_MFA_USER_ID     | No       | X |   | The user ID for the account to test in the Okta sandbox. Required only if running the live Okta MFA tests. |
| OKTA_MFA_USER_PASSWORD| No      | X |   | The password for the account to test in the Okta sandbox. Required only if running the live Okta MFA tests. |
| OKTA_MFA_SMS_FACTOR_ID | No     | X |   | The SMS MFA factor ID enrolled for the account to test in the Okta sandbox. Required only if running the live Okta MFA tests. |
| SSAS_DEFAULT_SYSTEM_SCOPE | Yes | X |   | Used to set the scope on systems that do not specify their scope. Must be set or runtime failures will occur. |
| SSAS_HASH_ITERATIONS | Yes      | X |   | Controls how many iterations our secure hashing mechanism performs. Service will panic if this key does not have a value. |
| SSAS_HASH_KEY_LENGTH | Yes      | X |   | Controls the key length used by our secure hashing mechanism. Service will panic if this key does not have a value. |
| SSAS_HASH_SALT_SIZE  | Yes      | X |   | Controls salt size used by our secure hashing mechanism performs. Service will panic if this key does not have a value. |
| SSAS_MFA_PROVIDER    | No       | X |   | Switches between mock Okta MFA calls and live calls.  Defaults to "Mock". |
| SSAS_MFA_CHALLENGE_ <br/> REQUEST_MILLISECONDS | No | X |   | Minimum execution time for RequestFactorChallenge().  If not present, defaults to 1500.  In production, this should always be set longer than the longest expected execution time.  (Actual execution time is logged.)|
| SSAS_MFA_TOKEN_ <br/> TIMEOUT_MINUTES | No | X |   | Token lifetime for self-registration (MFA tokens and Registration tokens).  Defaults to 60 (minutes). |
| SSAS_READ_TIMEOUT    | No       | X |   | Sets the read timeout on server requests |
| SSAS_WRITE_TIMEOUT   | No       | X |   | Sets the write timeout on server responses |
| SSAS_IDLE_TIMEOUT    | No       | X |   | Sets how long the server will keep idle connections open |
| SSAS_LOG                     | No  | X |   | Directs all ssas logging to a named file |
| SSAS_ADMIN_PORT <br/> SSAS_PUBLIC_PORT <br/> SSAS_HTTP_TO_HTTPS_PORT | No  | X | X | These values are not yet used by code. Intended to allow changing port assignments. If used, will affect BCDA SSAS URL vars. |
| SSAS_ADMIN_SIGNING_KEY_PATH  | Yes | X |   | Provides the location of the admin server signing key. When setting vars for AWS envs, you must include a var for the key material. |
| SSAS_PUBLIC_SIGNING_KEY_PATH | Yes | X |   | Provides the location of the public server signing key. When setting vars for AWS envs, you must include a var for the key material. |
| SSAS_TOKEN_BLACKLIST_CACHE_ <br/> CLEANUP_MINUTES  | No | X | | Tunes the frequency that expired entries are cleared from the token blacklist cache.  Defaults to 15 minutes. |
| SSAS_TOKEN_BLACKLIST_CACHE_ <br/> TIMEOUT_MINUTES  | No | X | | Sets the lifetime of token blacklist cache entries.  Defaults to 24 hours. |
| SSAS_TOKEN_BLACKLIST_CACHE_ <br/> REFRESH_MINUTES  | No | X | | Configures the number of minutes between times the token blacklist cache is refreshed from the database. |
| BCDA_TLS_CERT        | Depends  | X |   | The cert used when the SSAS service is running in secure mode. When setting vars for AWS envs, you must include a var for the cert material. This var should be renamed to SSAS_TLS_CERT. |
| BCDA_TLS_KEY         | Depends  | X |   | The private key used when the SSAS service is running in secure mode. When setting vars for AWS envs, you must include a var for the key material. This var should be renamed to SSAS_TLS_KEY. |

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
environmental variables. It is also possible to run individual tests, but that may require configurations for each test.

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
