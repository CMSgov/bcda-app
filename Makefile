package:
	# This target should be executed by passing in an argument reprsenting the version of the artifacts we are packaging
	# For example: make package version=r1
	docker build -t packaging -f Dockerfiles/Dockerfile.package .
	docker run -v ${PWD}:/go/src/github.com/CMSgov/bcda-app packaging $(version) 

test:
	docker-compose up -d db
	docker-compose -f docker-compose.test.yml up --force-recreate --exit-code-from unit_test

load-fixtures:
	docker-compose up -d db
	echo "Wait for db to be ready..."
	sleep 5
	docker-compose exec db psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -f /var/db/fixtures.sql

load-beneficiaries:
	docker-compose up -d db
	echo "Wait for db to be ready..."
	sleep 5
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

.PHONY: docker-build docker-bootstrap load-fixtures test debug-api debug-worker api-shell worker-shell package
