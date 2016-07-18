.PHONY: build run test clean

BINARY := router
SOURCE_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
BUILDFILES := router.go main.go router_api.go version.go
IMPORT_BASE := github.com/alphagov
IMPORT_PATH := $(IMPORT_BASE)/$(BINARY)

ifdef RELEASE_VERSION
VERSION := $(RELEASE_VERSION)
else
VERSION := $(shell git describe --always | tr -d '\n'; test -z "`git status --porcelain`" || echo '-dirty')
endif

build: $(BINARY)

run:
	go run $(BUILDFILES)

test: $(BINARY)
	go test ./trie ./triemux
	go test -v ./integration_tests

clean:
	rm -rf $(BINARY)

veryclean: clean
	rm -rf vendor

$(BINARY): $(SOURCE_FILES)
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) $(BUILDFILES)
