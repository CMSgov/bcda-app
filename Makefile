package:
	# This target should be executed by passing in an argument representing the version of the artifacts we are packaging
	# For example: make package version=r1
	docker-compose up documentation
	docker build -t packaging -f Dockerfiles/Dockerfile.package .
	docker run --rm \
	-e BCDA_GPG_RPM_PASSPHRASE='${BCDA_GPG_RPM_PASSPHRASE}' \
	-e GPG_RPM_USER='${GPG_RPM_USER}' \
	-e GPG_RPM_EMAIL='${GPG_RPM_EMAIL}' \
	-e GPG_PUB_KEY_FILE='${GPG_PUB_KEY_FILE}' \
	-e GPG_SEC_KEY_FILE='${GPG_SEC_KEY_FILE}' \
	-v ${PWD}:/go/src/github.com/CMSgov/bcda-app packaging $(version)

lint:
	docker-compose -f docker-compose.test.yml run --rm tests golangci-lint run --deadline=3m
	docker-compose -f docker-compose.test.yml run --rm tests gosec ./...

# The following vars are available to tests needing SSAS admin credentials; currently they are used in smoke-test
# Note that these variables should only be used for smoke tests, must be set before the api starts, and cannot be changed after the api starts
SSAS_ADMIN_CLIENT_ID ?= 31e029ef-0e97-47f8-873c-0e8b7e7f99bf
SSAS_ADMIN_CLIENT_SECRET := $(shell docker-compose run --rm ssas sh -c 'main --reset-secret --client-id=$(SSAS_ADMIN_CLIENT_ID)'|tail -n1)

#
# The following vars are used by both smoke-test and postman to pass credentials for obtaining an access token.
# The CLIENT_ID and CLIENT_SECRET values can be overridden by environmental variables e.g.:
#    export CLIENT_ID=1234; export CLIENT_SECRET=abcd; make postman env=local
# or 
#    CLIENT_ID=1234 CLIENT_SECRET=abcd make postman env=local
#
# If the values for CLIENT_ID and CLIENT_SECRET are not overridden, then by default, generate-client-credentials is
# called using ACO CMS ID "A9994" (to generate credentials for the `ACO Dev` which has a CMS ID of A9994 in our test
# data). This can be overridden using the same technique as above (exporting the env var and running make).
# For example:
#    export ACO_CMS_ID=A9999; make postman env=local
# or
#    ACO_CMS_ID=A9999 make postman env=local
#
ACO_CMS_ID ?= A9994
clientTemp := $(shell docker-compose run --rm api sh -c 'tmp/bcda reset-client-credentials --cms-id $(ACO_CMS_ID)'|tail -n2)
CLIENT_ID ?= $(shell echo $(clientTemp) |awk '{print $$1}')
CLIENT_SECRET ?= $(shell echo $(clientTemp) |awk '{print $$2}')
smoke-test:
	BCDA_SSAS_CLIENT_ID=$(SSAS_ADMIN_CLIENT_ID) BCDA_SSAS_SECRET=$(SSAS_ADMIN_CLIENT_SECRET) test/smoke_test/smoke_test.sh

postman:
	# This target should be executed by passing in an argument for the environment (dev/test/sbx)
	# and if needed a token.
	# Use env=local to bring up a local version of the app and test against it
	# For example: make postman env=test token=<MY_TOKEN>
	docker-compose -f docker-compose.test.yml run --rm postman_test test/postman_test/BCDA_Tests_Sequential.postman_collection.json -e test/postman_test/$(env).postman_environment.json --global-var "token=$(token)" --global-var clientId=$(CLIENT_ID) --global-var clientSecret=$(CLIENT_SECRET)

unit-test:
	$(MAKE) unit-test-db
	docker-compose -f docker-compose.test.yml run --rm tests bash unit_test.sh

unit-test-db:
	# Target stands up the postgres instance needed for unit testing.
	docker-compose -f docker-compose.test.yml rm -fsv db-test
	docker-compose -f docker-compose.test.yml up -d db-test

unit-test-db-snapshot:
	# Target takes a snapshot of the currently running postgres instance used for unit testing and updates the db/testing/docker-entrypoint-initdb.d/dump.pgdata file
	docker-compose -f docker-compose.test.yml exec db-test sh -c 'PGPASSWORD=$$POSTGRES_PASSWORD pg_dump -U postgres --format custom --file=/docker-entrypoint-initdb.d/dump.pgdata --create $$POSTGRES_DB'

performance-test:
	docker-compose -f docker-compose.test.yml run --rm -w /go/src/github.com/CMSgov/bcda-app/test/performance_test tests sh performance_test.sh

test:
	$(MAKE) lint
	$(MAKE) unit-test
	$(MAKE) postman env=local
	$(MAKE) smoke-test

load-fixtures:
	docker-compose up -d db
	echo "Wait for database to be ready..."
	sleep 5
	docker-compose run db psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -f /var/db/fixtures.sql
	$(MAKE) load-synthetic-cclf-data
	$(MAKE) load-synthetic-suppression-data
	$(MAKE) load-fixtures-ssas

load-synthetic-cclf-data:
	docker-compose up -d api
	docker-compose up -d db
	# The "test" environment provides baseline CCLF ingestion for ACO
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=dev --environment=test'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=dev-auth --environment=test'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=small --environment=test'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=medium --environment=test'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=large --environment=test'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=extra-large --environment=test'
	echo "Updating timestamp data on historical CCLF data for simulating ability to test /Group with _since"
	docker-compose run db psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "update cclf_files set timestamp='2020-02-01';"
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=dev --environment=test-new-beneficiaries'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=dev-auth --environment=test-new-beneficiaries'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=small --environment=test-new-beneficiaries'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=medium --environment=test-new-beneficiaries'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=large --environment=test-new-beneficiaries'
	docker-compose run api sh -c 'tmp/bcda import-synthetic-cclf-package --acoSize=extra-large --environment=test-new-beneficiaries'

load-synthetic-suppression-data:
	docker-compose up -d api
	docker-compose up -d db
	docker-compose run api sh -c 'tmp/bcda import-suppression-directory --directory=../shared_files/synthetic1800MedicareFiles'

load-fixtures-ssas:
	docker-compose -f docker-compose.ssas-migrate.yml run --rm ssas-migrate -database "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -path /go/src/github.com/CMSgov/bcda-ssas-app/db/migrations up
	docker-compose run ssas sh -c 'main --add-fixture-data'

docker-build:
	docker-compose build --force-rm
	docker-compose -f docker-compose.test.yml build --force-rm

docker-bootstrap:
	$(MAKE) docker-build
	docker-compose up -d
	sleep 40
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

bdt:
	# supply this target with the necessary environment vars, e.g.:
	# make bdt BDT_BASE_URL=<origin of API>
	docker build --no-cache -t bdt -f Dockerfiles/Dockerfile.bdt .
	docker run --rm \
	-e BASE_URL='${BDT_BASE_URL}' \
	-e CLIENT_ID='${CLIENT_ID}' \
	-e SECRET='${CLIENT_SECRET}' \
	bdt

.PHONY: api-shell debug-api debug-worker docker-bootstrap docker-build lint load-fixtures load-fixtures-ssas load-synthetic-cclf-data load-synthetic-suppression-data package performance-test postman release smoke-test test unit-test worker-shell bdt unit-test-db unit-test-db-snapshot
