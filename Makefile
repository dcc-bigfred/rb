.PHONY: all build test

all: build

build:
	CGO_ENABLED=0 GOOS=linux go build -o bin/rb .

test:
	go test ./...
