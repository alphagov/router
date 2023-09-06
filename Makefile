.PHONY: all clean build test lint unit_tests integration_tests start_mongo stop_mongo update_deps
.NOTPARALLEL:

TARGET_MODULE := router
GO_BUILD_ENV := CGO_ENABLED=0
SHELL := /bin/dash

all: build

clean:
	rm -f $(TARGET_MODULE)

build:
	env $(GO_BUILD_ENV) go build
	./$(TARGET_MODULE) -version

test: lint unit_tests integration_tests

lint:
	@if ! command -v golangci-lint; then \
		echo "linting uses golangci-lint: you can install it with:\n"; \
		echo "    brew install golangci-lint\n"; \
		exit 1; \
	fi
	golangci-lint run

unit_tests:
	go test -race $$(go list ./... | grep -v integration_tests)

integration_tests: build start_postgres init_test_db
	go test -race -v ./integration_tests

start_postgres:
	@if [ -z $$(docker ps -aqf name=router-postgres-test-db) ]; then \
		docker run \
			--name router-postgres-test-db \
			-e POSTGRES_HOST_AUTH_METHOD=trust \
			-d \
			-p 5432:5432 \
			--user 'postgres' \
			--health-cmd 'pg_isready' \
			--health-start-period 5s \
			postgres:14; \
		echo Waiting for postgres to be up; \
		for _ in $$(seq 60); do \
			if [ "$$(docker inspect -f '{{.State.Health.Status}}' router-postgres-test-db)" = "healthy" ]; then \
				break; \
			fi; \
			echo '.\c'; \
			sleep 1; \
		done; \
	else \
		echo "PostgreSQL container 'router-postgres-test-db' already exists. Skipping creation."; \
	fi

init_test_db:
	docker exec -i router-postgres-test-db psql < localdb_init.sql
	
cleanup_postgres:
	@docker rm -f router-postgres-test-db || true

update_deps:
	go get -t -u ./... && go mod tidy && go mod vendor
