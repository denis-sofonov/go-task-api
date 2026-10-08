-include .env
export

.PHONY: run build test test-race lint migrate migrate-down migrate-status

run:
	go run ./cmd/api

build:
	go build -o bin/ ./cmd/api

test:
	go test -cover ./...

test-race:
	go test -race -cover ./...

lint:
	golangci-lint run ./...

migrate:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status
