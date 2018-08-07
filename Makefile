models:
	PGSSLMODE=disable xo postgresql://postgres:toor@localhost:5432/bcda -o models

test:
	golangci-lint run
	docker-compose up -d db && sleep 5
	PGPASSWORD=toor psql -U postgres -h localhost -p 5432 -q -c 'drop database if exists bcda_test;'
	PGPASSWORD=toor psql -U postgres -h localhost -p 5432 -q -c 'create database bcda_test;'
	PGPASSWORD=toor psql -U postgres -h localhost -p 5432 -q -f db/api.sql bcda_test
	PGPASSWORD=toor psql -U postgres -h localhost -p 5432 -q -f db/fixtures.sql bcda_test
	DATABASE_URL="postgresql://postgres:toor@localhost:5432/bcda_test?sslmode=disable" go test -v -race ./...
	PGPASSWORD=toor psql -U postgres -h localhost -p 5432 -q -c 'drop database bcda_test;'

load-fixtures:
	docker-compose up -d db
	echo "Wait for db to be ready..."
	sleep 5
	psql "postgresql://postgres:toor@localhost:5432/bcda" -f db/fixtures.sql
	docker-compose stop db

docker-build:
	docker-compose build

docker-bootstrap: docker-build load-fixtures

.PHONY: models docker-build docker-bootstrap load-fixtures test
