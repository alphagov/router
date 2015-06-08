.PHONY: build run test clean

BINARY := router
BUILDFILES := router.go main.go router_api.go version.go
IMPORT_BASE := github.com/alphagov
IMPORT_PATH := $(IMPORT_BASE)/router

ifdef RELEASE_VERSION
VERSION := $(RELEASE_VERSION)
else
VERSION := $(shell git describe --always | tr -d '\n'; test -z "`git status --porcelain`" || echo '-dirty')
endif

build: _vendor/stamp
	gom build -ldflags "-X main.version $(VERSION)" -o $(BINARY) $(BUILDFILES)

run: _vendor/stamp
	gom run $(BUILDFILES)

test: _vendor/stamp build
	gom test ./trie ./triemux
	gom test -v ./integration_tests

clean:
	rm -f $(BINARY)

_vendor/stamp: Gomfile
	rm -f _vendor/src/$(IMPORT_PATH)
	mkdir -p _vendor/src/$(IMPORT_BASE)
	ln -s $(CURDIR) _vendor/src/$(IMPORT_PATH)
	gom install
	touch _vendor/stamp
