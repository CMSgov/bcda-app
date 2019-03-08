## Beneficiary Claims Data API

[![Build Status](https://travis-ci.org/CMSgov/bcda-app.svg?branch=master)](https://travis-ci.org/CMSgov/bcda-app)

### Dependencies

To get started, install some dependencies:

1. Install [Go](https://golang.org/doc/install)
2. Install [Docker](https://docs.docker.com/install/)
3. Install [Docker Compose](https://docs.docker.com/compose/install/)
4. Ensure all dependencies installed above are on PATH and can be executed directly from command line.

### Build / Start

Build the images and start the containers:

1. Build the images and load with fixture data
```sh
make docker-bootstrap
```

2. Start the containers
```sh
docker-compose up
```

### Test

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

### Use the application

See: [API documentation](https://github.com/CMSgov/bcda-app/blob/master/API.md)

### Handling secrets

To handle secrets safely, edit `.env.sh` to include any environment variables you need for local development.

From the `bcda-app/` directory, simply
```
$ cp env.sh .env.sh
```
make any changes to `.env.sh` to add any necessary environment variables and then
```
$ source .env.sh
```

This file is ignored by git and changes will not be tracked so you won't have to worry about exposing any secrets.

### Environment variables

Configure the `bcda` and `bcdaworker` apps by setting the following environment variables.

##### bcda

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

##### bcdaworker

```
BCDA_WORKER_ERROR_LOG <file_path>
BCDA_BB_LOG <file_path>
BB_CLIENT_CERT_FILE <file_path>
BB_CLIENT_KEY_FILE <file_path>
BB_SERVER_LOCATION <url>
FHIR_PAYLOAD_DIR <directory_path>
BB_TIMEOUT_MS <integer>
```

### Other things you can do

Use docker to look at the api database with psql:
```sh
docker run --rm --network bcda-app_default -it postgres psql -h bcda-app_db_1 -U postgres bcda
```

See docker-compose.yml for the password.

Use docker to run the CLI against an API instance
```
docker exec -it bcda-app_api_1 bash -c 'tmp/bcda -h'
```

If you have no data in your database, you can load the fixture data with
```sh
make load-fixtures
```
