.PHONY: build run test clean

BINARY := router
SOURCE_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
BUILDFILES := router.go main.go router_api.go version.go
IMPORT_BASE := github.com/alphagov
IMPORT_PATH := $(IMPORT_BASE)/$(BINARY)

# Set a GOPATH to prevent gom erroring.
GOPATH := $(CURDIR)/gopath
export GOPATH

ifdef RELEASE_VERSION
VERSION := $(RELEASE_VERSION)
else
VERSION := $(shell git describe --always | tr -d '\n'; test -z "`git status --porcelain`" || echo '-dirty')
endif

build: $(BINARY)

run: vendor/stamp
	gom run $(BUILDFILES)

test: $(BINARY)
	gom test ./trie ./triemux
	gom test -v ./integration_tests

clean:
	rm -rf $(BINARY) vendor

$(BINARY): $(SOURCE_FILES) vendor/stamp
	gom build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) $(BUILDFILES)

vendor/stamp: Gomfile
	gom install
	ln -s $(CURDIR)/vendor vendor/src
	rm -f vendor/src/$(IMPORT_PATH)
	mkdir -p vendor/src/$(IMPORT_BASE)
	ln -s $(CURDIR) vendor/src/$(IMPORT_PATH)
	touch vendor/stamp
