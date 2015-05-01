.PHONY: build run test clean

BINARY := router
BUILDFILES := router.go main.go router_api.go
IMPORT_BASE := github.com/alphagov
IMPORT_PATH := $(IMPORT_BASE)/router

build: _vendor/stamp
	gom build -o $(BINARY) $(BUILDFILES)

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
