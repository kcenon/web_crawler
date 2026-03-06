.PHONY: all build test lint proto clean

BINARY := bin/crawler
PROTO_DIR := api/proto

all: lint test build

build:
	go build -o $(BINARY) ./cmd/crawler

test:
	go test ./...

lint:
	golangci-lint run ./...

proto:
	cd $(PROTO_DIR) && buf lint
	cd $(PROTO_DIR) && buf generate

clean:
	rm -rf bin/
