.PHONY: all build test lint proto clean

BINARY := bin/crawler

all: lint test build

build:
	go build -o $(BINARY) ./cmd/crawler

test:
	go test ./...

lint:
	golangci-lint run ./...

proto:
	@echo "proto generation not yet configured"

clean:
	rm -rf bin/
