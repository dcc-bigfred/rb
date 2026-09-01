.PHONY: all build test dist

all: build

build:
	CGO_ENABLED=0 GOOS=linux go build -o bin/rb .

test:
	go test ./...

dist:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/rb-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/rb-linux-arm64 .
