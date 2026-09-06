.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -o bin/beehiiv-pp-cli$(BIN_EXT) ./cmd/beehiiv-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/beehiiv-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/beehiiv-pp-mcp$(BIN_EXT) ./cmd/beehiiv-pp-mcp

install-mcp:
	go install ./cmd/beehiiv-pp-mcp

build-all: build build-mcp
