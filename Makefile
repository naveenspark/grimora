.PHONY: build install test lint check

build:
	go build -o grimora ./cmd/grimora/

install:
	go install -ldflags "-s -w" ./cmd/grimora/

test:
	go test -race ./...

lint:
	golangci-lint run

check: lint test
