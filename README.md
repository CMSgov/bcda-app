## Beneficiary Claims Data API

[![Build Status](https://travis-ci.org/CMSgov/bcda-app.svg?branch=master)](https://travis-ci.org/CMSgov/bcda-app)

### Local development

To get started:

1. Install [Go](https://golang.org/doc/install)
2. Install [Docker](https://docs.docker.com/install/)
3. Install [Docker Compose](https://docs.docker.com/compose/install/)
4. Install [xo](https://github.com/xo/xo)
5. Install [usql](https://github.com/xo/usql)
6. Ensure all dependencies installed above are on PATH and can be executed directly from command line.

```sh
make docker-bootstrap
docker-compose up
```

To interact with the app:
1) Get a token: `http://localhost:3000/api/v1/token`
2) POST to `http://localhost:3000/api/v1/claims` with `Authorization: Bearer <token>`


To run tests:
```
make test
```
