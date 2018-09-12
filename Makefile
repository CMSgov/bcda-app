models:
	docker-compose up -d db
	echo "Waiting for db to be ready..."
	sleep 5	
	PGSSLMODE=disable xo postgresql://postgres:toor@localhost:5432/bcda -o models

test:
	docker-compose up -d db
	docker-compose -f docker-compose.test.yml up --force-recreate

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

.PHONY: models docker-build docker-bootstrap load-fixtures test
