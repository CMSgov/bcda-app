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

1. Build the images
```sh
make docker-bootstrap
```
2. Start the containers
```sh
docker-compose up
```

### Test

Run tests and produce test metrics:

1. Run tests (this places results and a coverage report in test_results/<timestamp>):
```sh
make test
```

### Use the application

See: [API documentation](https://github.com/CMSgov/bcda-app/blob/master/API.md)

### Environment variables

Configure the `bcda` and `bcdaworker` apps by setting the following environment variables.

##### bcda

```
BCDA_ERROR_LOG <file_path>
BCDA_REQUEST_LOG <file_path>
BCDA_BB_LOG <file_path>
BB_CLIENT_CERT_FILE <file_path>
BB_CLIENT_KEY_FILE <file_path>
BB_SERVER_LOCATION <url>
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
