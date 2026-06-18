.PHONY: build test lint tidy

build:
	go build -o bin/psy ./cmd/psy

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy
