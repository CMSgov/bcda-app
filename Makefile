package:
	# This target should be executed by passing in an argument representing the version of the artifacts we are packaging
	# For example: make package version=r1
	docker-compose up -d documentation
	docker-compose up -d static_site
	docker build -t packaging -f Dockerfiles/Dockerfile.package .
	docker run --rm \
	-e BCDA_GPG_RPM_PASSPHRASE='${BCDA_GPG_RPM_PASSPHRASE}' \
	-e GPG_RPM_USER='${GPG_RPM_USER}' \
	-e GPG_RPM_EMAIL='${GPG_RPM_EMAIL}' \
	-e GPG_PUB_KEY_FILE='${GPG_PUB_KEY_FILE}' \
	-e GPG_SEC_KEY_FILE='${GPG_SEC_KEY_FILE}' \
	-v ${PWD}:/go/src/github.com/CMSgov/bcda-app packaging $(version)

smoke-test:
	docker-compose up -d 
	sleep 30
	docker-compose -f docker-compose.test.yml up --build --force-recreate --exit-code-from smoke_test smoke_test

unit-test:
	docker-compose up -d db queue
	docker-compose -f docker-compose.test.yml up --build --force-recreate --exit-code-from unit_test unit_test

postman:
	# This target should be executed by passing in an argument for the environment (dev/test/sbx)
	# and if needed a token.
	# Use env=local to bring up a local version of the app and test against it
	# For example: make postman env=test token=<MY_TOKEN>
ifeq ($(env), local)
	docker-compose up -d
	sleep 30
endif
	docker-compose -f docker-compose.test.yml build --no-cache postman_test
	docker-compose -f docker-compose.test.yml run --rm postman_test test/postman_test/$(env).postman_environment.json --global-var "token=$(token)"

performance-test:
	docker-compose up -d
	sleep 30
	docker-compose -f docker-compose.test.yml up --build --force-recreate --exit-code-from performance_test performance_test

test:
	$(MAKE) unit-test
	$(MAKE) postman env=local
	$(MAKE) smoke-test

load-fixtures:
	docker-compose up -d db
	echo "Wait for db to be ready..."
	sleep 5
	docker-compose exec db psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -f /var/db/fixtures.sql
	docker-compose exec db psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -f /var/db/synthetic_beneficiaries.sql

docker-build:
	docker-compose build
	docker-compose -f docker-compose.test.yml build

docker-bootstrap: docker-build load-fixtures

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

.PHONY: docker-build docker-bootstrap load-fixtures test debug-api debug-worker api-shell worker-shell package release smoke-test postman unit-test performance-test
