.PHONY: build test unit_tests integration_tests clean start_mongo clean_mongo clean_mongo_again show_metrics

BINARY ?= router
SHELL := /bin/bash

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

test: start_mongo unit_tests integration_tests clean_mongo_again

unit_tests: build
	go test -race $$(go list ./... | grep -v integration_tests)

integration_tests: start_mongo build
	ROUTER_PUBADDR=localhost:8080 \
		go test -race -v ./integration_tests

start_mongo: clean_mongo
	@if ! docker run --rm --name router-mongo -dp 27017:27017 mongo:2.4 --replSet rs0 --quiet; then \
		echo 'Failed to start mongo; if using Docker Desktop, try:' ; \
		echo ' - disabling Settings -> Features in development -> Use containerd' ; \
		echo ' - enabling Settings -> Features in development -> Use Rosetta' ; \
		exit 1 ; \
	fi
	@echo -n Waiting for mongo
	@for n in {1..30}; do \
		if docker exec router-mongo mongo --quiet --eval 'rs.initiate()' >/dev/null 2>&1; then \
			sleep 1; \
			echo ; \
			break ; \
		fi ; \
		echo -n . ; \
		sleep 1 ; \
	done ; \

clean_mongo clean_mongo_again:
	docker rm -f router-mongo >/dev/null 2>&1 || true
	@sleep 1  # Docker doesn't queue commands so it races with itself :(

show_metrics:
	@git grep 'Name\|Help' `# this generate awful output when there is no Help defined for a metric` \
		| grep -v vendor  `# do not look for vendored metrics` \
		| grep metrics.go `# we only define our metrics here` \
		| sed 's/.*: //'  `# do not look at the filename from git grep` \
		| tr -d ',"' \
		| awk 'NR%2 == 1 { s=50-length($$0) ; $$0 = sprintf("%s%" sprintf("%s", s) "s", $$0, "") }1' `# right pad the help text`\
		| sed 'N;s/\n/ /'              `# put help text and metric name on same line` \
		| sort

check_metrics: build clean_mongo start_mongo
	$(BINARY) & echo "$$$$!" > /tmp/router.pid
	@until [ "`curl -s -o /dev/null -w '%{http_code}' http://localhost:8081/metrics`" = "200" ]; do \
		echo '.\c'  ; \
	  sleep 1     ; \
	done          ; \
	echo
	(curl -sf http://localhost:8081/metrics ; &> /dev/null kill $$(cat /tmp/router.pid)) | promtool check metrics
