.PHONY: build run test clean set_local_env start_postgres cleanup_postgres show_metrics

BINARY ?= $(PWD)/router-postgres

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
	go test -race $$(go list ./... | grep -v integration_tests)
	go test -race -v ./integration_tests

run: build
	$(BINARY)

set_local_env:
	@echo Setting listen addr to be localhost and debug to be true
	$(eval export ROUTER_PUBADDR ?= 127.0.0.1:8080)
	$(eval export DEBUG ?= true)

start_postgres:
	docker network create router-postgres-test-db || true
	docker run -dit \
		         --name router-postgres-test-db \
				 		 -e POSTGRES_HOST_AUTH_METHOD=trust \
						 -d \
						 -p 5432:5432 \
						 --user 'postgres' \
						 --health-cmd 'pg_isready' \
						 --health-start-period 5s \
						 --network router-postgres-test-db \
						 postgres:14
	@echo Waiting for postgres to be up
	@until [ "`docker inspect -f '{{.State.Health.Status}}' router-postgres-test-db`" = "healthy" ]; do \
		echo '.\c'  ; \
	  sleep 1     ; \
	done          ;

CONTAINER_NAME ?= govuk-docker_postgres-14_1
setup_local_db:
	docker exec -i $(CONTAINER_NAME) psql -c "CREATE DATABASE router;" && \
	docker exec -i $(CONTAINER_NAME) psql -d router < localdb_init.sql

cleanup_postgres:
	@docker rm -f router-postgres-test-db || true

test_with_docker: $(eval CONTAINER_NAME := router-postgres-test-db) cleanup_postgres start_postgres setup_local_db set_local_env test

show_metrics:
	@git grep 'Name\|Help' `# this generate awful output when there is no Help defined for a metric` \
		| grep -v vendor  `# do not look for vendored metrics` \
		| grep metrics.go `# we only define our metrics here` \
		| sed 's/.*: //'  `# do not look at the filename from git grep` \
		| tr -d ',"' \
		| awk 'NR%2 == 1 { s=50-length($$0) ; $$0 = sprintf("%s%" sprintf("%s", s) "s", $$0, "") }1' `# right pad the help text`\
		| sed 'N;s/\n/ /'              `# put help text and metric name on same line` \
		| sort

check_metrics: build cleanup_mongo start_mongo
	$(BINARY) & echo "$$$$!" > /tmp/router.pid
	@until [ "`curl -s -o /dev/null -w '%{http_code}' http://localhost:8081/metrics`" = "200" ]; do \
		echo '.\c'  ; \
	  sleep 1     ; \
	done          ; \
	echo
	(curl -sf http://localhost:8081/metrics ; &> /dev/null kill $$(cat /tmp/router.pid)) | promtool check metrics
