.PHONY: all clean build lint test unit_tests integration_tests start_mongo stop_mongo
.NOTPARALLEL:

BINARY ?= router
SHELL := /bin/dash

ifdef RELEASE_VERSION
VERSION := $(RELEASE_VERSION)
else
VERSION := $(shell git describe --always | tr -d '\n'; test -z "`git status --porcelain`" || echo '-dirty')
endif

all: build test

clean:
	rm -f $(BINARY)

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY)

lint:
	golangci-lint run

test: lint unit_tests integration_tests

unit_tests: build
	go test -race $$(go list ./... | grep -v integration_tests)

integration_tests: build start_mongo
	ROUTER_PUBADDR=localhost:8080 \
	ROUTER_APIADDR=localhost:8081 \
		go test -race -v ./integration_tests

start_mongo:
	./mongo.sh start

stop_mongo:
	./mongo.sh stop
