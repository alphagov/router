.PHONY: build run test clean

BINARY := router
BUILDFILES := router.go main.go router_api.go
IMPORT_BASE := github.com/alphagov
IMPORT_PATH := $(IMPORT_BASE)/router

build: _vendor
	gom build -o $(BINARY) $(BUILDFILES)

run: _vendor
	gom run $(BUILDFILES)

test: _vendor build
	gom test ./trie ./triemux
	gom test -v ./integration_tests
	bundle exec rspec

clean:
	rm -f $(BINARY)

_vendor: Gomfile _vendor/src/$(IMPORT_PATH)
	gom install
	touch _vendor

_vendor/src/$(IMPORT_PATH):
	rm -f _vendor/src/$(IMPORT_PATH)
	mkdir -p _vendor/src/$(IMPORT_BASE)
	ln -s $(CURDIR) _vendor/src/$(IMPORT_PATH)
