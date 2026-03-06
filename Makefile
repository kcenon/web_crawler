.PHONY: all build test lint proto clean

BINARY := bin/crawler
PROTO_DIR := api/proto

all: proto lint test build

build: proto
	go build -o $(BINARY) ./cmd/crawler

test: proto
	go test ./...

lint: proto
	golangci-lint run ./...

proto:
	cd $(PROTO_DIR) && buf lint
	cd $(PROTO_DIR) && buf generate

clean:
	rm -rf bin/
