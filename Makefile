.PHONY: build run test clean

BINARY ?= $(PWD)/router

ifdef RELEASE_VERSION
VERSION := $(RELEASE_VERSION)
else
VERSION := $(shell git describe --always | tr -d '\n'; test -z "`git status --porcelain`" || echo '-dirty')
endif

all: clean build test

clean:
	rm -rf $(BINARY)

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY)

test: build
	go test -race ./trie ./triemux
	go test -race -v ./integration_tests

run: build
	$(BINARY)
