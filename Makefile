.PHONY: build run test clean set_local_env start_mongo cleanup_mongo

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

set_local_env:
	@echo Setting listen addr to be localhost and debug to be true
	$(eval export ROUTER_PUBADDR ?= 127.0.0.1:8080)
	$(eval export DEBUG ?= true)

start_mongo:
	docker run -dit \
		         --name router-mongo \
						 -d \
						 -p 27017:27017 \
						 --health-cmd 'curl localhost:27017' \
						 --health-start-period 15s \
						 mongo:2.4.11
	@echo Waiting for mongo to be up
	@until [ "`docker inspect -f '{{.State.Health.Status}}' router-mongo`" = "healthy" ]; do \
		echo '.\c'  ; \
	  sleep 1     ; \
	done          ; \
	echo

cleanup_mongo:
	@docker rm -f router-mongo || true

test_with_docker: cleanup_mongo start_mongo set_local_env test
