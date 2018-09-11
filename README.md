## Beneficiary Claims Data API

[![Build Status](https://travis-ci.org/CMSgov/bcda-app.svg?branch=master)](https://travis-ci.org/CMSgov/bcda-app)


### Dependencies

To get started, install some dependencies:

1. Install [Go](https://golang.org/doc/install)
2. Install [Docker](https://docs.docker.com/install/)
3. Install [Docker Compose](https://docs.docker.com/compose/install/)
4. Install [xo](https://github.com/xo/xo)
5. Ensure all dependencies installed above are on PATH and can be executed directly from command line.


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

To interact with the app:

1) Get a token: `http://localhost:3000/api/v1/token`
2) POST to `http://localhost:3000/api/v1/claims` with `Authorization: Bearer <token>`
