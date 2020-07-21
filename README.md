# Beneficiary Claims Data API

[![Build Status](https://travis-ci.org/CMSgov/bcda-app.svg?branch=master)](https://travis-ci.org/CMSgov/bcda-app)

## Dependencies

To get started, install some dependencies:

1. Install [Go](https://golang.org/doc/install)
2. Install [Docker](https://docs.docker.com/install/)
3. Install [Docker Compose](https://docs.docker.com/compose/install/)
4. Install [Ansible Vault](https://docs.ansible.com/ansible/2.4/vault.html)
5. Ensure all dependencies installed above are on PATH and can be executed directly from command line.

## Sensitive Docker Configuration Files

The files committed in the `shared_files/encrypted` directory hold secret information, and are encrypted with [Ansible Vault](https://docs.ansible.com/ansible/2.4/vault.html).

### Setup
#### Password
- See a team member for the Ansible Vault password
- Create a file named `.vault_password` in the root directory of the repository
- Place the Ansible Vault password in this file

#### Git hook
To avoid committing and pushing unencrypted secret files, use the included `scripts/pre-commit` git pre-commit hook by running the following script from the repository root directory:
```
cp ops/pre-commit .git/hooks
```
The pre-commit hook will also ensure that any added, copied, or modified go files are formatted properly.

### Managing encrypted files
* Temporarily decrypt files by running the following command from the repository root directory: 
```
./ops/secrets --decrypt
```
* While files are decrypted, copy the files in this directory to the sibling directory `shared_files/decrypted`
* Encrypt changed files with:
```
./ops/secrets --encrypt <filename>
```  

## Build / Start

Build the images and start the containers:

1. Build the images and load with fixture data
```sh
make docker-bootstrap
```

2. Start the containers
```sh
docker-compose up
```

## Test

Run tests and produce test metrics.  
The items identified above in the `Build/Start` section are prerequisites to running tests.  
In order to keep the test feedback loop optimized, the following items must be handled by the caller (and are not handled by the test targets):
- Ensuring the compose stack is up and running
- Ensuring the database has been seeded
- Managing images/containers (if Dockerfile changes have occurred, an image rebuild is required and won't occur as part of the test targets)

1. Run golang linter and gosec:
```sh
make lint
```

2. Run unit tests (this places results and a coverage report in test_results/<timestamp>):
```sh
make unit-test
```

3. Run postman integration tests:
```sh
make postman env=local
```

4. Run smoke tests:
```sh
make smoke-test
```

5. Run full test suite (executes all of items in 1-4 above):
```sh
make test
```

6. Run performance tests (primarily to be utilized by Jenkins in AWS):
```sh
make performance-test
```

### Updating seed data for unit tests

After the user has finished updating the Postgres db used for unit testing with the new data, the user can update
the seed data by running the following comamnd:
```sh
make unit-test-db-snapshot
```

This script will update `./db/testing/docker-entrypoint-initdb.d/dump.pgdata` file.
This file is used to initialize the Postgres db with all of the necessary data needed for the various unit tests.

For more information on intialization, please see `db/testing/docker-entrypoint-initdb.d/01-restore.sh`.
This script is executed when the Postgres container is launched.

The updated `dump.pgdata` should be committed with the other associated changes.

## Use the application

See: [API documentation](https://bcda.cms.gov/sandbox/user-guide/)

## Handling secrets

### **NEVER PUT PASSWORDS, KEYS, OR SECRETS OF ANY KIND IN APPLICATION CODE!  INSTEAD, USE THE STRATEGY OUTLINED HERE**

In the project root `bcda-app/` directory, create a file called `.env.sh`. This file is ignored by git and will not be committed
```
$ touch .env.sh
```

Next, edit `.env.sh` to include the bash shebang and any necessary environment variables like this
```
#!/bin/bash
export BCDA_AUTH_PROVIDER=okta
export OKTA_OAUTH_SERVER_ID="<serverID>"
export OKTA_CLIENT_TOKEN="<apiKey>"
export OKTA_CLIENT_SECRET="<clientSecret>"
```

Lastly, source the file to add the variables to your local development environment
```
$ source .env.sh
```

You're good to go! Use the environment variables in application code like this
```
apiKey := os.Getenv("OKTA_CLIENT_TOKEN")
```

Optionally, you can edit your `~/.zshrc` or `~/.bashrc` file to eliminate the need to source the file for each shell start by appending this line
```
source $GOPATH/src/github.com/CMSgov/bcda-app/.env.sh
```

## Environment variables

Configure the `bcda` and `bcdaworker` apps by setting the following environment variables.

### bcda

```
BCDA_ERROR_LOG <file_path>
BCDA_REQUEST_LOG <file_path>
BCDA_BB_LOG <file_path>
BCDA_OKTA_LOG <file_path>
BB_CLIENT_CERT_FILE <file_path>
BB_CLIENT_KEY_FILE <file_path>
BB_SERVER_LOCATION <url>
OKTA_CLIENT_TOKEN <api_key>
OKTA_CLIENT_ORGURL <url>
OKTA_EMAIL <test_account>
FHIR_PAYLOAD_DIR <directory_path>
JWT_EXPIRATION_DELTA <integer> (time in hours that JWT access tokens are valid for)
```

### bcdaworker

```
BCDA_WORKER_ERROR_LOG <file_path>
BCDA_BB_LOG <file_path>
BB_CLIENT_CERT_FILE <file_path>
BB_CLIENT_KEY_FILE <file_path>
BB_SERVER_LOCATION <url>
FHIR_PAYLOAD_DIR <directory_path>
BB_TIMEOUT_MS <integer>
```

## Other things you can do

Use docker to look at the api database with psql:
```sh
docker run --rm --network bcda-app_default -it postgres psql -h bcda-app_db_1 -U postgres bcda
```

See docker-compose.yml for the password.

Use docker to run the CLI against an API instance
```
docker exec -it bcda-app_api_1 sh -c 'tmp/bcda -h'
```

If you have no data in your database, you can load the fixture data with
```sh
make load-fixtures
```
