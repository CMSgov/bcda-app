decrypt-secrets:
# check for ansible in path
	@which ansible-vault > /dev/null || (echo "ansible-vault not found; ansible-vault must be installed to decrypt secrets" ; exit 1)

# check for vault password file
	@[ -f .vault_password ] || (echo "vault password not found; ensure .vault_password file exists at repository root" ; exit 1)

	@echo "Decrypt and overwrite local secrets? (y/N)";
	@read response; \
	if [[ "$$response" == "y" || "$$response" == "Y" ]]; then \
		./ops/secrets --decrypt ; \
		cp -a ./shared_files/encrypted/. ./shared_files/decrypted/ ; \
		git checkout ./shared_files/encrypted/ ; \
	else \
		echo "Operation cancelled." ; \
	fi

setup-tests:
	# Clean up any existing data to ensure we spin up container in a known state.
	docker compose -f docker-compose.test.yml rm -fsv tests
	docker compose -f docker-compose.test.yml build tests

LINT_TIMEOUT ?= 5m
lint: setup-tests
	docker compose -f docker-compose.test.yml run --rm --remove-orphans \
		tests golangci-lint run --timeout=$(LINT_TIMEOUT) --verbose --new-from-merge-base=main
	# TODO: Remove the exclusion of G301 as part of BCDA-8414
	docker compose -f docker-compose.test.yml run --rm tests gosec -exclude=G301 ./... ./optout

smoke-test: setup-tests
	test/smoke_test/smoke_test.sh $(env)

postman:
	# This target should be executed by passing in an argument for the environment (dev/test/sandbox)
	# and if needed a token.
	# Use env=local to bring up a local version of the app and test against it
	# For example: make postman env=test token=<MY_TOKEN>

	echo $(env)
	@if test -z "$(env)"; then \
		echo "Error: postman target must include an 'env' argument (e.g. 'env=local')"; \
		exit 1; \
	fi

	$(eval BLACKLIST_CLIENT_ID=$(shell docker compose exec -T api env | grep BLACKLIST_CLIENT_ID | cut -d'=' -f2))
	$(eval BLACKLIST_CLIENT_SECRET=$(shell docker compose exec -T api env | grep BLACKLIST_CLIENT_SECRET | cut -d'=' -f2))

	# Set up valid client credentials
	$(eval ACO_CMS_ID = A9994)
	$(eval CLIENT_TEMP := $(shell docker compose run --rm api sh -c 'bcda reset-client-credentials --cms-id $(ACO_CMS_ID)'|tail -n2))
	$(eval CLIENT_ID:=$(shell echo $(CLIENT_TEMP) |awk '{print $$1}'))
	$(eval CLIENT_SECRET:=$(shell echo $(CLIENT_TEMP) |awk '{print $$2}'))

	# Set up valid client credentials for outdated attribution client
	$(eval OUTDATED_ATTR_CMS_ID = TEST995)
	$(eval OUTDATED_ATTR_CLIENT_TEMP := $(shell docker compose run --rm api sh -c 'bcda reset-client-credentials --cms-id $(OUTDATED_ATTR_CMS_ID)'|tail -n2))
	$(eval OUTDATED_ATTR_CLIENT_ID:=$(shell echo $(OUTDATED_ATTR_CLIENT_TEMP) |awk '{print $$1}'))
	$(eval OUTDATED_ATTR_CLIENT_SECRET:=$(shell echo $(OUTDATED_ATTR_CLIENT_TEMP) |awk '{print $$2}'))

	docker compose -f docker-compose.test.yml build postman_test
	@docker compose -f docker-compose.test.yml run --rm postman_test test/postman_test/BCDA_Tests_Sequential.postman_collection.json \
	-e test/postman_test/$(env).postman_environment.json --global-var "token=$(token)" --global-var clientId=$(CLIENT_ID) --global-var clientSecret=$(CLIENT_SECRET) \
	--global-var blacklistedClientId=$(BLACKLIST_CLIENT_ID) --global-var blacklistedClientSecret=$(BLACKLIST_CLIENT_SECRET) \
	--global-var outdatedAttrClientId=$(OUTDATED_ATTR_CLIENT_ID) --global-var outdatedAttrClientSecret=$(OUTDATED_ATTR_CLIENT_SECRET) \
	--global-var v2Disabled=false

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
# 	docker run --rm --network bcda-app-net willwill/wait-for-it db-unit-test:5432 -t 120
	sleep 100

	# Perform migrations to ensure matching schemas
	docker run --rm -v ${PWD}/db/migrations:/migrations --network bcda-app-net migrate/migrate -path=/migrations/bcda/ -database 'postgres://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable&x-migrations-table=schema_migrations_bcda' up

unit-test-localstack:
	# Clean up any existing data to ensure we spin up container in a known state.
	docker compose -f docker-compose.test.yml rm -fsv localstack-unit-test
	docker compose -f docker-compose.test.yml up -d localstack-unit-test

unit-test-db-snapshot:
	# Target takes a snapshot of the currently running postgres instance used for unit testing and updates the db/testing/docker-entrypoint-initdb.d/dump.pgdata file
	docker compose -f docker-compose.test.yml exec db-unit-test sh -c 'PGPASSWORD=$$POSTGRES_PASSWORD pg_dump -U postgres --format custom --file=/docker-entrypoint-initdb.d/dump.pgdata --create $$POSTGRES_DB'

performance-test: setup-tests
	docker compose -f docker-compose.test.yml run --rm -w /go/src/github.com/CMSgov/bcda-app/test/performance_test tests sh performance_test.sh

test:
	$(MAKE) lint
	$(MAKE) unit-test
	$(MAKE) postman env=local
	$(MAKE) smoke-test env=local

reset-db:
	# Rebuild the databases to ensure that we're starting in a fresh state
	docker compose -f docker-compose.yml rm -fsv db

	docker compose up -d db
	echo "Wait for database to be ready..."
# 	docker run --rm --network bcda-app-net willwill/wait-for-it db:5432 -t 100
	sleep 100

	# Initialize schemas
	docker run --rm -v ${PWD}/db/migrations:/migrations --network bcda-app-net migrate/migrate -path=/migrations/bcda/ -database 'postgres://postgres:toor@db:5432/bcda?sslmode=disable&x-migrations-table=schema_migrations_bcda' up

load-fixtures: reset-db
# Start api service if it's not already running
	docker compose up -d api
	$(MAKE) load-fixtures-ssas

	# Ensure components are started as expected
	docker compose up -d api worker ssas
# 	docker run --rm --network bcda-app-net willwill/wait-for-it api:3000 -t 30
# 	docker run --rm --network bcda-app-net willwill/wait-for-it ssas:3003 -t 30
	sleep 30
	docker compose run --rm db psql -v ON_ERROR_STOP=1 "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -f /var/db/bootstrap.sql

load-fixtures-ssas:
	docker compose up -d db
	docker run --rm --network bcda-app-net migrate/migrate:v4.15.0-beta.3 -source='github://CMSgov/bcda-ssas-app/db/migrations#main' -database 'postgres://postgres:toor@db:5432/bcda?sslmode=disable' up
	docker compose run --rm ssas --add-fixture-data

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

fhir_testing:
	# Set up inferno server
	docker build -t inferno:1 https://github.com/inferno-framework/bulk-data-test-kit.git#5bd61db090c5911792f33e12dca6981d7e22f9a0
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

generate-mocks:
	docker run -v "$PWD":/src -w /src vektra/mockery:v3.6.1

.PHONY: api-shell debug-api debug-worker docker-bootstrap docker-build generate-mocks lint load-fixtures load-fixtures-ssas package performance-test postman release smoke-test test unit-test worker-shell bdt fhir_testing unit-test-db unit-test-db-snapshot reset-db dbdocs

credentials:
	$(eval ACO_CMS_ID = A9994)
	# Use ACO_CMS_ID to generate a local set of credentials for the ACO.
	# For example: ACO_CMS_ID=A9993 make credentials
	@docker compose run --rm api sh -c 'bcda reset-client-credentials --cms-id $(ACO_CMS_ID)'|tail -n2

dbdocs:
	docker run --rm -v $PWD:/work -w /work --network bcda-app-net ghcr.io/k1low/tbls doc --rm-dist "postgres://postgres:toor@db:5432/bcda?sslmode=disable" dbdocs/bcda

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
# make build-api ACCOUNT_ID={AWS account ID} RELEASE_VERSION={release tag eg r270 (or main)}
build-api:
	$(eval ACCOUNT_ID =$(shell aws sts get-caller-identity --output text --query Account))
	$(eval CURRENT_COMMIT=$(shell git log -n 1 --pretty=format:'%h'))
	$(eval DOCKER_REGISTRY_URL=${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/bcda-api)
	docker build -t ${DOCKER_REGISTRY_URL}:latest -t '${DOCKER_REGISTRY_URL}:${RELEASE_VERSION}' -f Dockerfiles/Dockerfile.bcda .

publish-api:
	$(eval ACCOUNT_ID =$(shell aws sts get-caller-identity --output text --query Account))
	$(eval DOCKER_REGISTRY_URL=${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/bcda-api)
	aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin '${DOCKER_REGISTRY_URL}'
	docker image push '${DOCKER_REGISTRY_URL}' -a

build-worker:
	$(eval ACCOUNT_ID =$(shell aws sts get-caller-identity --output text --query Account))
	$(eval CURRENT_COMMIT=$(shell git log -n 1 --pretty=format:'%h'))
	$(eval DOCKER_REGISTRY_URL=${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/bcda-worker)
	docker build -t ${DOCKER_REGISTRY_URL}:latest -t '${DOCKER_REGISTRY_URL}:${RELEASE_VERSION}' -f Dockerfiles/Dockerfile.bcdaworker .

publish-worker:
	$(eval ACCOUNT_ID =$(shell aws sts get-caller-identity --output text --query Account))
	$(eval DOCKER_REGISTRY_URL=${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/bcda-worker)
	aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin '${DOCKER_REGISTRY_URL}'
	docker image push '${DOCKER_REGISTRY_URL}' -a
