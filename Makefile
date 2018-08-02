models:
	PGSSLMODE=disable xo postgresql://postgres:toor@localhost:5432/bcda -o models

test:
	golangci-lint run
	go test -v -race ./...

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
