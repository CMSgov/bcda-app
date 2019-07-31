package:
	# This target should be executed by passing in an argument representing the version of the artifacts we are packaging
	# For example: make package version=r1
	docker-compose up documentation
	docker-compose up static_site
	docker build -t packaging -f Dockerfiles/Dockerfile.package .
	docker run --rm \
	-e BCDA_GPG_RPM_PASSPHRASE='${BCDA_GPG_RPM_PASSPHRASE}' \
	-e GPG_RPM_USER='${GPG_RPM_USER}' \
	-e GPG_RPM_EMAIL='${GPG_RPM_EMAIL}' \
	-e GPG_PUB_KEY_FILE='${GPG_PUB_KEY_FILE}' \
	-e GPG_SEC_KEY_FILE='${GPG_SEC_KEY_FILE}' \
	-v ${PWD}:/go/src/github.com/CMSgov/bcda-app packaging $(version)

lint:
	docker-compose -f docker-compose.test.yml run --rm tests golangci-lint run 
	docker-compose -f docker-compose.test.yml run --rm tests gosec ./...

lint-ssas:
	docker-compose -f docker-compose.test.yml run --rm tests golangci-lint run ./ssas/...
	docker-compose -f docker-compose.test.yml run --rm tests gosec ./ssas/...

#
# The following vars are used by both smoke-test and postman to pass credentials for obtaining an access token
# The CLIENT_ID and CLIENT_SECRET values can be overridden by environmental variables e.g.:
#    export CLIENT_ID=1234; export CLIENT_SECRET=abcd; make postman env=local
# or 
#    CLIENT_ID=1234 CLIENT_SECRET=abcd make postman env=local
#
# If the values for CLIENT_ID and CLIENT_SECRET are not overwritten, then by default, generate-client-credentials is called using ACO CMS ID "A9994" (to generate credentials for the `ACO Dev` which has a CMS ID of A9994 in our test data)
# This can be overridden using the same technique above (exporting the env var and running make).
# For example:
#    export ACO_CMS_ID=A9999; make postman env=local
# or
#    ACO_CMS_ID=A9999 make postman env=local
#
ACO_CMS_ID ?= A9994
clientTemp := $(shell docker-compose run --rm api sh -c 'tmp/bcda generate-client-credentials --cms-id $(ACO_CMS_ID)'|tail -n2)
CLIENT_ID ?= $(shell echo $(clientTemp) |awk '{print $$1}')
CLIENT_SECRET ?= $(shell echo $(clientTemp) |awk '{print $$2}')
smoke-test:
	CLIENT_ID=$(CLIENT_ID) CLIENT_SECRET=$(CLIENT_SECRET) docker-compose -f docker-compose.test.yml run --rm -w /go/src/github.com/CMSgov/bcda-app/test/smoke_test tests sh smoke_test.sh 

postman:
	# This target should be executed by passing in an argument for the environment (dev/test/sbx)
	# and if needed a token.
	# Use env=local to bring up a local version of the app and test against it
	# For example: make postman env=test token=<MY_TOKEN>
	docker-compose -f docker-compose.test.yml run --rm postman_test test/postman_test/$(env).postman_environment.json --global-var "token=$(token)" --global-var clientId=$(CLIENT_ID) --global-var clientSecret=$(CLIENT_SECRET)

unit-test:
	docker-compose -f docker-compose.test.yml run --rm tests bash unit_test.sh

unit-test-ssas:
	docker-compose -f docker-compose.test.yml run --rm tests bash unit_test_ssas.sh

performance-test:
	docker-compose -f docker-compose.test.yml run --rm -w /go/src/github.com/CMSgov/bcda-app/test/performance_test tests sh performance_test.sh

test:
	$(MAKE) lint
	$(MAKE) unit-test
	$(MAKE) postman env=local
	$(MAKE) smoke-test

test-ssas:
	$(MAKE) lint-ssas
	$(MAKE) unit-test-ssas
	@echo "No postman tests (yet)"
	@echo "No smoke-test (yet)"

load-fixtures:
	docker-compose up -d db
	echo "Wait for database to be ready..."
	sleep 5
	docker-compose run db psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -f /var/db/fixtures.sql
	$(MAKE) load-synthetic-cclf-data
	$(MAKE) load-synthetic-suppression-data

load-synthetic-cclf-data:
	docker-compose up -d api
	docker-compose up -d db
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=dev --environment=test'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=small --environment=test'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=medium --environment=test'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=large --environment=test'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=extra-large --environment=test'

load-synthetic-suppression-data:
	docker-compose up -d api
	docker-compose up -d db
	docker-compose run api sh -c 'tmp/bcda import-suppression-directory --directory=../shared_files/synthetic1800MedicareFiles'

docker-build:
	docker-compose build --force-rm
	docker-compose -f docker-compose.test.yml build --force-rm

docker-bootstrap:
	$(MAKE) docker-build
	docker-compose up -d
	sleep 30
	$(MAKE) load-fixtures

api-shell:
	docker-compose exec api bash

worker-shell:
	docker-compose exec worker bash

debug-api:
	docker-compose start db queue worker
	@echo "Starting debugger. This may take a while..."
	@-bash -c "trap 'docker-compose stop' EXIT; \
		docker-compose -f docker-compose.yml -f docker-compose.debug.yml run --no-deps -T --rm -p 3000:3000 -v $(shell pwd):/go/src/github.com/CMSgov/bcda-app api dlv debug -- start-api"

debug-worker:
	docker-compose start db queue api
	@echo "Starting debugger. This may take a while..."
	@-bash -c "trap 'docker-compose stop' EXIT; \
		docker-compose -f docker-compose.yml -f docker-compose.debug.yml run --no-deps -T --rm -v $(shell pwd):/go/src/github.com/CMSgov/bcda-app worker dlv debug"

.PHONY: docker-build docker-bootstrap load-fixtures load-synthetic-cclf-data load-synthetic-suppression-data test debug-api debug-worker api-shell worker-shell package release smoke-test postman unit-test performance-test lint
