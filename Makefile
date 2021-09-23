package:
	# This target should be executed by passing in an argument representing the version of the artifacts we are packaging
	# For example: make package version=r1
	docker-compose up --build documentation
	docker-compose up --build openapi
	docker build -t packaging -f Dockerfiles/Dockerfile.package .
	docker run --rm \
	-e BCDA_GPG_RPM_PASSPHRASE='${BCDA_GPG_RPM_PASSPHRASE}' \
	-e GPG_RPM_USER='${GPG_RPM_USER}' \
	-e GPG_RPM_EMAIL='${GPG_RPM_EMAIL}' \
	-e GPG_PUB_KEY_FILE='${GPG_PUB_KEY_FILE}' \
	-e GPG_SEC_KEY_FILE='${GPG_SEC_KEY_FILE}' \
	-v ${PWD}:/go/src/github.com/CMSgov/bcda-app packaging $(version)


LINT_TIMEOUT ?= 3m
lint:
	docker-compose -f docker-compose.test.yml build tests
	docker-compose -f docker-compose.test.yml run \
	--rm tests golangci-lint run --exclude="(conf\.(Un)?[S,s]etEnv)" --deadline=$(LINT_TIMEOUT) --verbose
	docker-compose -f docker-compose.test.yml run --rm tests gosec ./...

smoke-test:
	docker-compose -f docker-compose.test.yml build tests
	test/smoke_test/smoke_test.sh

postman:
	# This target should be executed by passing in an argument for the environment (dev/test/sbx)
	# and if needed a token.
	# Use env=local to bring up a local version of the app and test against it
	# For example: make postman env=test token=<MY_TOKEN>
	$(eval BLACKLIST_CLIENT_ID=$(shell docker-compose exec -T api env | grep BLACKLIST_CLIENT_ID | cut -d'=' -f2))
	$(eval BLACKLIST_CLIENT_SECRET=$(shell docker-compose exec -T api env | grep BLACKLIST_CLIENT_SECRET | cut -d'=' -f2))

	# Set up valid client credentials
	$(eval ACO_CMS_ID = A9994)
	$(eval CLIENT_TEMP := $(shell docker-compose run --rm api sh -c 'bcda reset-client-credentials --cms-id $(ACO_CMS_ID)'|tail -n2))
	$(eval CLIENT_ID:=$(shell echo $(CLIENT_TEMP) |awk '{print $$1}'))
	$(eval CLIENT_SECRET:=$(shell echo $(CLIENT_TEMP) |awk '{print $$2}'))

	docker-compose -f docker-compose.test.yml build postman_test
	@docker-compose -f docker-compose.test.yml run --rm postman_test test/postman_test/BCDA_Tests_Sequential.postman_collection.json \
	-e test/postman_test/$(env).postman_environment.json --global-var "token=$(token)" --global-var clientId=$(CLIENT_ID) --global-var clientSecret=$(CLIENT_SECRET) \
	--global-var blacklistedClientId=$(BLACKLIST_CLIENT_ID) --global-var blacklistedClientSecret=$(BLACKLIST_CLIENT_SECRET) \
	--global-var v2Disabled=false \
	--global-var alrEnabled=true

unit-test:
	$(MAKE) unit-test-db
	
	docker-compose -f docker-compose.test.yml build tests
	@docker-compose -f docker-compose.test.yml run --rm tests bash scripts/unit_test.sh

unit-test-db:
	# Target stands up the postgres instance needed for unit testing.

	# Clean up any existing data to ensure we spin up container in a known state.
	docker-compose -f docker-compose.test.yml rm -fsv db-unit-test
	docker-compose -f docker-compose.test.yml up -d db-unit-test
	
	# Wait for the database to be ready
	docker run --rm --network bcda-app-net willwill/wait-for-it db-unit-test:5432 -t 120
	
	# Perform migrations to ensure matching schemas
	docker run --rm -v ${PWD}/db/migrations:/migrations --network bcda-app-net migrate/migrate -path=/migrations/bcda/ -database 'postgres://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable&x-migrations-table=schema_migrations_bcda' up
	docker run --rm -v ${PWD}/db/migrations:/migrations --network bcda-app-net migrate/migrate -path=/migrations/bcda_queue/ -database 'postgres://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable&x-migrations-table=schema_migrations_bcda_queue' up

	# Load ALR data into the unit-test-DB for local and github actions unit-test
	# TODO: once we finalize on synthetic ALR data, we should take a snapshot of the unit-test-db
	docker-compose run \
	-e DATABASE_URL=postgresql://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable \
	-e QUEUE_DATABASE_URL=postgresql://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable \
	api sh -c 'bcda generate-synthetic-alr-data --cms-id=A9994 --alr-template-file ./alr/gen/testdata/PY21ALRTemplatePrelimProspTable1.csv'
	
unit-test-db-snapshot:
	# Target takes a snapshot of the currently running postgres instance used for unit testing and updates the db/testing/docker-entrypoint-initdb.d/dump.pgdata file
	docker-compose -f docker-compose.test.yml exec db-unit-test sh -c 'PGPASSWORD=$$POSTGRES_PASSWORD pg_dump -U postgres --format custom --file=/docker-entrypoint-initdb.d/dump.pgdata --create $$POSTGRES_DB'

performance-test:
	docker-compose -f docker-compose.test.yml build tests
	docker-compose -f docker-compose.test.yml run --rm -w /go/src/github.com/CMSgov/bcda-app/test/performance_test tests sh performance_test.sh

test:
	$(MAKE) lint
	$(MAKE) unit-test
	$(MAKE) postman env=local
	$(MAKE) smoke-test

load-fixtures:
	# Rebuild the databases to ensure that we're starting in a fresh state
	docker-compose -f docker-compose.yml rm -fsv db queue

	docker-compose up -d db queue
	echo "Wait for databases to be ready..."
	docker run --rm --network bcda-app-net willwill/wait-for-it db:5432 -t 100
	docker run --rm --network bcda-app-net willwill/wait-for-it queue:5432 -t 100

	# Initialize schemas
	docker run --rm -v ${PWD}/db/migrations:/migrations --network bcda-app-net migrate/migrate -path=/migrations/bcda/ -database 'postgres://postgres:toor@db:5432/bcda?sslmode=disable&x-migrations-table=schema_migrations_bcda' up
	docker run --rm -v ${PWD}/db/migrations:/migrations --network bcda-app-net migrate/migrate -path=/migrations/bcda_queue/ -database 'postgres://postgres:toor@queue:5432/bcda_queue?sslmode=disable&x-migrations-table=schema_migrations_bcda_queue' up

	docker-compose run db psql -v ON_ERROR_STOP=1 "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -f /var/db/fixtures.sql
	$(MAKE) load-synthetic-cclf-data
	$(MAKE) load-synthetic-suppression-data
	$(MAKE) load-fixtures-ssas
	
	# Add ALR data for ACOs under test. Must have attribution already set.
	$(eval ACO_CMS_IDS := A9994 A9996) 
	for acoId in $(ACO_CMS_IDS) ; do \
		docker-compose run api sh -c 'bcda generate-synthetic-alr-data --cms-id='$$acoId' --alr-template-file ./alr/gen/testdata/PY21ALRTemplatePrelimProspTable1.csv' ; \
	done

	# Ensure components are started as expected
	docker-compose up -d api worker ssas
	docker run --rm --network bcda-app-net willwill/wait-for-it api:3000 -t 100
	docker run --rm --network bcda-app-net willwill/wait-for-it ssas:3003 -t 100

	# Additional fixtures for postman+ssas
	docker-compose run db psql -v ON_ERROR_STOP=1 "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -f /var/db/postman_fixtures.sql

load-synthetic-cclf-data:
	$(eval ACO_SIZES := dev dev-auth dev-cec dev-cec-auth dev-ng dev-ng-auth dev-ckcc dev-ckcc-auth dev-kcf dev-kcf-auth dev-dc dev-dc-auth small medium large extra-large)
	# The "test" environment provides baseline CCLF ingestion for ACO
	docker-compose run --rm api sh -c "../scripts/bulk_import_synthetic_cclf_package.sh test ' ' $(ACO_SIZES)"
	echo "Updating timestamp data on historical CCLF data for simulating ability to test /Group with _since"
	docker-compose run db psql -v ON_ERROR_STOP=1 "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "update cclf_files set timestamp='2020-02-01';"
	docker-compose run --rm api sh -c "../scripts/bulk_import_synthetic_cclf_package.sh test-new-beneficiaries ' ' $(ACO_SIZES)"
	docker-compose run --rm api sh -c "../scripts/bulk_import_synthetic_cclf_package.sh test runout $(ACO_SIZES)"

load-synthetic-suppression-data:
	docker-compose run api sh -c 'bcda import-suppression-directory --directory=../shared_files/synthetic1800MedicareFiles'
	# Update the suppression entries to guarantee there are qualified entries when searching for suppressed benes.
	# See postgres#GetSuppressedMBIs for more information
	docker-compose exec -T db sh -c 'PGPASSWORD=$$POSTGRES_PASSWORD psql -v ON_ERROR_STOP=1 $$POSTGRES_DB postgres -c "UPDATE suppressions SET effective_date = now(), preference_indicator = '"'"'N'"'"'  WHERE effective_date = (SELECT max(effective_date) FROM suppressions);"'

load-fixtures-ssas:
	docker-compose up -d db
	docker run --rm --network bcda-app-net migrate/migrate:v4.15.0-beta.3 -source='github://CMSgov/bcda-ssas-app/db/migrations#master' -database 'postgres://postgres:toor@db:5432/bcda?sslmode=disable' up
	docker-compose run ssas --add-fixture-data

docker-build:
	docker-compose build --force-rm
	docker-compose -f docker-compose.test.yml build --force-rm

docker-bootstrap: docker-build documentation load-fixtures

api-shell:
	docker-compose exec -T api bash

worker-shell:
	docker-compose exec -T worker bash

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
	# Set up valid client credentials
	$(eval ACO_CMS_ID = A9994)
	$(eval CLIENT_TEMP := $(shell docker-compose run --rm api sh -c 'bcda reset-client-credentials --cms-id $(ACO_CMS_ID)'|tail -n2))
	$(eval CLIENT_ID:=$(shell echo $(CLIENT_TEMP) |awk '{print $$1}'))
	$(eval CLIENT_SECRET:=$(shell echo $(CLIENT_TEMP) |awk '{print $$2}'))
	$(eval BDT_BASE_URL = 'http://host.docker.internal:3000')
	docker build --no-cache -t bdt -f Dockerfiles/Dockerfile.bdt .
	@docker run --rm \
	-e BASE_URL='${BDT_BASE_URL}' \
	-e CLIENT_ID='${CLIENT_ID}' \
	-e SECRET='${CLIENT_SECRET}' \
	bdt

.PHONY: api-shell debug-api debug-worker docker-bootstrap docker-build lint load-fixtures load-fixtures-ssas load-synthetic-cclf-data load-synthetic-suppression-data package performance-test postman release smoke-test test unit-test worker-shell bdt unit-test-db unit-test-db-snapshot

documentation:
	docker-compose up --build documentation
	docker-compose up --exit-code-from openapi openapi

credentials:
	$(eval ACO_CMS_ID = A9994)
	# Use ACO_CMS_ID to generate a local set of credentials for the ACO.
	# For example: ACO_CMS_ID=A9993 make credentials 
	@docker-compose run --rm api sh -c 'bcda reset-client-credentials --cms-id $(ACO_CMS_ID)'|tail -n2
