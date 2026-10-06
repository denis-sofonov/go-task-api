-include .env
export

.PHONY: run build test test-race lint

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
