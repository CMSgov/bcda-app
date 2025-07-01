package:
	# This target should be executed by passing in an argument representing the version of the artifacts we are packaging
	# For example: make package version=r1
	docker build -t packaging -f Dockerfiles/Dockerfile.package .
	docker run --rm \
	-e BCDA_GPG_RPM_PASSPHRASE='${BCDA_GPG_RPM_PASSPHRASE}' \
	-e GPG_RPM_USER='${GPG_RPM_USER}' \
	-e GPG_RPM_EMAIL='${GPG_RPM_EMAIL}' \
	-e GPG_PUB_KEY_FILE='${GPG_PUB_KEY_FILE}' \
	-e GPG_SEC_KEY_FILE='${GPG_SEC_KEY_FILE}' \
	-v ${PWD}:/go/src/github.com/CMSgov/bcda-app packaging $(version)

setup-tests:
	# Clean up any existing data to ensure we spin up container in a known state.
	docker compose -f docker-compose.test.yml rm -fsv tests
	docker compose -f docker-compose.test.yml build tests

# -D(isabling) errcheck, staticcheck, and govet linters for now due to v2 upgrade, see: https://jira.cms.gov/browse/BCDA-8911
LINT_TIMEOUT ?= 3m
lint: setup-tests
	docker compose -f docker-compose.test.yml run \
	--rm tests golangci-lint run --timeout=$(LINT_TIMEOUT) --verbose
	# TODO: Remove the exclusion of G301 as part of BCDA-8414
	docker compose -f docker-compose.test.yml run --rm tests gosec -exclude=G301 ./... ./optout

smoke-test: setup-tests
	test/smoke_test/smoke_test.sh $(env) $(maintenanceMode)

postman:
	# This target should be executed by passing in an argument for the environment (dev/test/sandbox)
	# and if needed a token.
	# Use env=local to bring up a local version of the app and test against it
	# For example: make postman env=test token=<MY_TOKEN> maintenanceMode=<CURRENT_MAINTENANCE_MODE>
	$(eval BLACKLIST_CLIENT_ID=$(shell docker compose exec -T api env | grep BLACKLIST_CLIENT_ID | cut -d'=' -f2))
	$(eval BLACKLIST_CLIENT_SECRET=$(shell docker compose exec -T api env | grep BLACKLIST_CLIENT_SECRET | cut -d'=' -f2))

	# Set up valid client credentials
	$(eval ACO_CMS_ID = A9994)
	$(eval CLIENT_TEMP := $(shell docker compose run --rm api sh -c 'bcda reset-client-credentials --cms-id $(ACO_CMS_ID)'|tail -n2))
	$(eval CLIENT_ID:=$(shell echo $(CLIENT_TEMP) |awk '{print $$1}'))
	$(eval CLIENT_SECRET:=$(shell echo $(CLIENT_TEMP) |awk '{print $$2}'))

	# to test alrEnabled, include --global-var alrEnabled=true below
	docker compose -f docker-compose.test.yml build postman_test
	@docker compose -f docker-compose.test.yml run --rm postman_test test/postman_test/BCDA_Tests_Sequential.postman_collection.json \
	-e test/postman_test/$(env).postman_environment.json --global-var "token=$(token)" --global-var clientId=$(CLIENT_ID) --global-var clientSecret=$(CLIENT_SECRET) \
	--global-var blacklistedClientId=$(BLACKLIST_CLIENT_ID) --global-var blacklistedClientSecret=$(BLACKLIST_CLIENT_SECRET) \
	--global-var v2Disabled=false \
	--global-var maintenanceMode=$(maintenanceMode)

# make test-path TEST_PATH="bcdaworker/worker/*.go"
test-path: setup-tests
	@docker compose -f docker-compose.test.yml run --rm tests go test -v $(TEST_PATH)

unit-test: unit-test-ssas unit-test-db unit-test-localstack load-fixtures-ssas setup-tests
	@docker compose -f docker-compose.test.yml run --rm tests bash scripts/unit_test.sh

unit-test-ssas:
	docker compose up -d ssas

unit-test-db:
	# Target stands up the postgres instance needed for unit testing.

	# Clean up any existing data to ensure we spin up container in a known state.
	docker compose -f docker-compose.test.yml rm -fsv db-unit-test
	docker compose -f docker-compose.test.yml up -d db-unit-test

	# Wait for the database to be ready
	docker run --rm --network bcda-app-net willwill/wait-for-it db-unit-test:5432 -t 120

	# Perform migrations to ensure matching schemas
	docker run --rm -v ${PWD}/db/migrations:/migrations --network bcda-app-net migrate/migrate -path=/migrations/bcda/ -database 'postgres://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable&x-migrations-table=schema_migrations_bcda' up
	docker run --rm -v ${PWD}/db/migrations:/migrations --network bcda-app-net migrate/migrate -path=/migrations/bcda_queue/ -database 'postgres://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable&x-migrations-table=schema_migrations_bcda_queue' up

	# Load ALR data into the unit-test-DB for local and github actions unit-test
	# TODO: once we finalize on synthetic ALR data, we should take a snapshot of the unit-test-db
	docker compose run \
	-e DATABASE_URL=postgresql://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable \
	-e QUEUE_DATABASE_URL=postgresql://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable \
	api sh -c 'bcda generate-synthetic-alr-data --cms-id=A9994 --alr-template-file ./alr/gen/testdata/PY21ALRTemplatePrelimProspTable1.csv'

unit-test-localstack:
	# Clean up any existing data to ensure we spin up container in a known state.
	docker compose -f docker-compose.test.yml rm -fsv localstack
	docker compose -f docker-compose.test.yml up -d localstack

unit-test-db-snapshot:
	# Target takes a snapshot of the currently running postgres instance used for unit testing and updates the db/testing/docker-entrypoint-initdb.d/dump.pgdata file
	docker compose -f docker-compose.test.yml exec db-unit-test sh -c 'PGPASSWORD=$$POSTGRES_PASSWORD pg_dump -U postgres --format custom --file=/docker-entrypoint-initdb.d/dump.pgdata --create $$POSTGRES_DB'

performance-test: setup-tests
	docker compose -f docker-compose.test.yml run --rm -w /go/src/github.com/CMSgov/bcda-app/test/performance_test tests sh performance_test.sh

test:
	$(MAKE) lint
	$(MAKE) unit-test
	$(MAKE) postman env=local maintenanceMode=""
	$(MAKE) smoke-test env=local maintenanceMode=""

reset-db:
	# Rebuild the databases to ensure that we're starting in a fresh state
	docker compose -f docker-compose.yml rm -fsv db queue

	docker compose up -d db queue
	echo "Wait for databases to be ready..."
	docker run --rm --network bcda-app-net willwill/wait-for-it db:5432 -t 100
	docker run --rm --network bcda-app-net willwill/wait-for-it queue:5432 -t 100

	# Initialize schemas
	docker run --rm -v ${PWD}/db/migrations:/migrations --network bcda-app-net migrate/migrate -path=/migrations/bcda/ -database 'postgres://postgres:toor@db:5432/bcda?sslmode=disable&x-migrations-table=schema_migrations_bcda' up
	docker run --rm -v ${PWD}/db/migrations:/migrations --network bcda-app-net migrate/migrate -path=/migrations/bcda_queue/ -database 'postgres://postgres:toor@queue:5432/bcda_queue?sslmode=disable&x-migrations-table=schema_migrations_bcda_queue' up

load-fixtures: reset-db
	docker compose run db psql -v ON_ERROR_STOP=1 "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -f /var/db/fixtures.sql
	$(MAKE) load-synthetic-cclf-data
	$(MAKE) load-synthetic-suppression-data
	$(MAKE) load-fixtures-ssas

	# Add ALR data for ACOs under test. Must have attribution already set.
	$(eval ACO_CMS_IDS := A9994 A9996)
	for acoId in $(ACO_CMS_IDS) ; do \
		docker compose run api sh -c 'bcda generate-synthetic-alr-data --cms-id='$$acoId' --alr-template-file ./alr/gen/testdata/PY21ALRTemplatePrelimProspTable1.csv' ; \
	done

	# Ensure components are started as expected
	docker compose up -d api worker ssas
	docker run --rm --network bcda-app-net willwill/wait-for-it api:3000 -t 30
	docker run --rm --network bcda-app-net willwill/wait-for-it ssas:3003 -t 30

	# Additional fixtures for postman+ssas
	docker compose run db psql -v ON_ERROR_STOP=1 "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -f /var/db/postman_fixtures.sql

load-synthetic-cclf-data:
	$(eval ACO_SIZES := dev dev-auth dev-cec dev-cec-auth dev-ng dev-ng-auth dev-ckcc dev-ckcc-auth dev-kcf dev-kcf-auth dev-dc dev-dc-auth small medium large extra-large)
	# The "test" environment provides baseline CCLF ingestion for ACO
	for ACO_SIZE in $(ACO_SIZES) ; do \
		docker compose run --rm api sh -c "bcda import-synthetic-cclf-package --acoSize='$$ACO_SIZE' --environment='test' --fileType='' " ; \
	done
	echo "Updating timestamp data on historical CCLF data for simulating ability to test /Group with _since"
	docker compose run db psql -v ON_ERROR_STOP=1 "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "update cclf_files set timestamp='2020-02-01';"
	for ACO_SIZE in $(ACO_SIZES) ; do \
		docker compose run --rm api sh -c "bcda import-synthetic-cclf-package --acoSize='$$ACO_SIZE' --environment='test-new-beneficiaries' --fileType='' " ; \
		docker compose run --rm api sh -c "bcda import-synthetic-cclf-package --acoSize='$$ACO_SIZE' --environment='test' --fileType='runout' " ; \
	done

	# Improved Synthea BFD Data Ingestion
	$(eval IMPROVED_SIZES := improved-dev improved-small improved-large)
	for IMPROVED_SIZE in $(IMPROVED_SIZES) ; do \
			docker compose run --rm api sh -c "bcda import-synthetic-cclf-package --acoSize='$$IMPROVED_SIZE' --environment='improved' --fileType='' " ; \
			docker compose run --rm api sh -c "bcda import-synthetic-cclf-package --acoSize='$$IMPROVED_SIZE' --environment='improved-new' --fileType='' " ; \
			docker compose run --rm api sh -c "bcda import-synthetic-cclf-package --acoSize='$$IMPROVED_SIZE' --environment='improved' --fileType='runout' " ; \
	done

load-synthetic-suppression-data:
	docker compose run api sh -c 'bcda import-suppression-directory --directory=../shared_files/synthetic1800MedicareFiles'
	# Update the suppression entries to guarantee there are qualified entries when searching for suppressed benes.
	# See postgres#GetSuppressedMBIs for more information
	docker compose exec -T db sh -c 'PGPASSWORD=$$POSTGRES_PASSWORD psql -v ON_ERROR_STOP=1 $$POSTGRES_DB postgres -c "UPDATE suppressions SET effective_date = now(), preference_indicator = '"'"'N'"'"'  WHERE effective_date = (SELECT max(effective_date) FROM suppressions);"'

load-fixtures-ssas:
	docker compose up -d db
	docker run --rm --network bcda-app-net migrate/migrate:v4.15.0-beta.3 -source='github://CMSgov/bcda-ssas-app/db/migrations#main' -database 'postgres://postgres:toor@db:5432/bcda?sslmode=disable' up
	docker compose run ssas --add-fixture-data

docker-build:
	docker compose build --force-rm
	docker compose -f docker-compose.test.yml build --force-rm

docker-bootstrap: docker-build load-fixtures

api-shell:
	docker compose exec -T api bash

worker-shell:
	docker compose exec -T worker bash

debug-api:
	docker compose up --watch worker & \
	docker compose -f docker-compose.yml -f docker-compose.debug.yml up --watch api

debug-worker:
	docker compose up --watch api & \
	docker compose -f docker-compose.yml -f docker-compose.debug.yml up --watch worker

bdt:
	# Set up valid client credentials
	$(eval ACO_CMS_ID = A9994)
	$(eval CLIENT_TEMP := $(shell docker compose run --rm api sh -c 'bcda reset-client-credentials --cms-id $(ACO_CMS_ID)'|tail -n2))
	$(eval CLIENT_ID:=$(shell echo $(CLIENT_TEMP) |awk '{print $$1}'))
	$(eval CLIENT_SECRET:=$(shell echo $(CLIENT_TEMP) |awk '{print $$2}'))
	$(eval BDT_BASE_URL = 'http://host.docker.internal:3000')
	docker build --no-cache -t bdt -f Dockerfiles/Dockerfile.bdt .
	@docker run --rm \
	-e BASE_URL='${BDT_BASE_URL}' \
	-e CLIENT_ID='${CLIENT_ID}' \
	-e SECRET='${CLIENT_SECRET}' \
	bdt

fhir_testing:
	# Set up inferno server
	docker build -t inferno:1 https://github.com/inferno-framework/bulk-data-test-kit.git
	docker compose -f fhir_testing/docker-compose.inferno.yml run inferno bundle exec inferno migrate
	docker compose -f fhir_testing/docker-compose.inferno.yml up -d
	sleep 10
	docker stop fhir_testing-hl7_validator_service-1

	# Get config
	$(eval ACO_CMS_ID = A9994)
	$(eval CLIENT_TEMP := $(shell docker compose run --rm api sh -c 'bcda reset-client-credentials --cms-id $(ACO_CMS_ID)'|tail -n2))
	$(eval CLIENT_ID:=$(shell echo $(CLIENT_TEMP) |awk '{print $$1}'))
	$(eval CLIENT_SECRET:=$(shell echo $(CLIENT_TEMP) |awk '{print $$2}'))
	$(eval BULK_URL = 'http://host.docker.internal:3000/api/v2/')
	$(eval TOKEN_URL = 'http://host.docker.internal:3000/auth/token')
	
	# Run the tests
	docker build --no-cache -t fhir_testing -f Dockerfiles/Dockerfile.fhir_testing .
	@docker run --network=bridge --rm \
	-e BULK_URL='${BULK_URL}' \
	-e TOKEN_URL='${TOKEN_URL}' \
	-e CLIENT_ID='${CLIENT_ID}' \
	-e CLIENT_SECRET='${CLIENT_SECRET}' \
	fhir_testing

.PHONY: api-shell debug-api debug-worker docker-bootstrap docker-build lint load-fixtures load-fixtures-ssas load-synthetic-cclf-data load-synthetic-suppression-data package performance-test postman release smoke-test test unit-test worker-shell bdt fhir_testing unit-test-db unit-test-db-snapshot reset-db dbdocs

credentials:
	$(eval ACO_CMS_ID = A9994)
	# Use ACO_CMS_ID to generate a local set of credentials for the ACO.
	# For example: ACO_CMS_ID=A9993 make credentials
	@docker compose run --rm api sh -c 'bcda reset-client-credentials --cms-id $(ACO_CMS_ID)'|tail -n2

dbdocs:
	docker run --rm -v $PWD:/work -w /work --network bcda-app-net ghcr.io/k1low/tbls doc --rm-dist "postgres://postgres:toor@db:5432/bcda?sslmode=disable" dbdocs/bcda
	docker run --rm -v $PWD:/work -w /work --network bcda-app-net ghcr.io/k1low/tbls doc --force "postgres://postgres:toor@queue:5432/bcda_queue?sslmode=disable" dbdocs/bcda_queue

# ==== Lambda ====

package-opt-out: export GOOS=linux
package-opt-out: export GOARCH=amd64
package-opt-out:
	cd bcda && go build -o bin/opt-out-import ./lambda/optout/main.go

package-cclf-import: export GOOS=linux
package-cclf-import: export GOARCH=amd64
package-cclf-import:
	cd bcda && go build -o bin/cclf-import ./lambda/cclf/main.go

# Build and publish images to ECR
build-api:
	$(eval ACCOUNT_ID =$(shell aws sts get-caller-identity --output text --query Account))
	$(eval CURRENT_COMMIT=$(shell git log -n 1 --pretty=format:'%h'))
	$(eval DOCKER_REGISTRY_URL=${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/bcda-api)
	docker build -t ${DOCKER_REGISTRY_URL}:latest -t '${DOCKER_REGISTRY_URL}:${CURRENT_COMMIT}' -f Dockerfiles/Dockerfile.bcda_prod .

publish-api:
	$(eval ACCOUNT_ID =$(shell aws sts get-caller-identity --output text --query Account))
	$(eval DOCKER_REGISTRY_URL=${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/bcda-api)
	aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin '${DOCKER_REGISTRY_URL}'
	docker image push '${DOCKER_REGISTRY_URL}' -a

build-worker:
	$(eval ACCOUNT_ID =$(shell aws sts get-caller-identity --output text --query Account))
	$(eval CURRENT_COMMIT=$(shell git log -n 1 --pretty=format:'%h'))
	$(eval DOCKER_REGISTRY_URL=${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/bcda-worker)
	docker build -t ${DOCKER_REGISTRY_URL}:latest -t '${DOCKER_REGISTRY_URL}:${CURRENT_COMMIT}' -f Dockerfiles/Dockerfile.bcdaworker_prod .

publish-worker:
	$(eval ACCOUNT_ID =$(shell aws sts get-caller-identity --output text --query Account))
	$(eval DOCKER_REGISTRY_URL=${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/bcda-worker)
	aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin '${DOCKER_REGISTRY_URL}'
	docker image push '${DOCKER_REGISTRY_URL}' -a
