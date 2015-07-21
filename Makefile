.PHONY: build run test clean

BINARY := router
SOURCE_FILES := $(shell find . -name '*.go' -not -path './_vendor/*')
BUILDFILES := router.go main.go router_api.go version.go
IMPORT_BASE := github.com/alphagov
IMPORT_PATH := $(IMPORT_BASE)/router

ifdef RELEASE_VERSION
VERSION := $(RELEASE_VERSION)
else
VERSION := $(shell git describe --always | tr -d '\n'; test -z "`git status --porcelain`" || echo '-dirty')
endif

build: $(BINARY)

run: _vendor/stamp
	gom run $(BUILDFILES)

test: $(BINARY)
	gom test ./trie ./triemux
	gom test -v ./integration_tests

clean:
	rm -rf $(BINARY) _vendor

$(BINARY): $(SOURCE_FILES) _vendor/stamp
	gom build -ldflags "-X main.version $(VERSION)" -o $(BINARY) $(BUILDFILES)

_vendor/stamp: Gomfile
	rm -f _vendor/src/$(IMPORT_PATH)
	mkdir -p _vendor/src/$(IMPORT_BASE)
	ln -s $(CURDIR) _vendor/src/$(IMPORT_PATH)
	gom install
	touch _vendor/stamp
