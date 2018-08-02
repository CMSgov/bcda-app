models:
	PGSSLMODE=disable xo postgresql://postgres:toor@localhost:5432/bcda -o models

test:
	golangci-lint run
	go test -v -race ./...

.PHONY: models
