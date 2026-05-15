.PHONY: all lint test build clean

all: lint test build

lint:
	golangci-lint run

test:
	go test -v -race ./...

build: build-client build-server

build-client:
	mkdir -p bin
	go build -o bin/client ./cmd/client

build-server:
	mkdir -p bin
	go build -o bin/server ./cmd/server

clean:
	rm -rf bin/
