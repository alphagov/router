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

integration_tests: build start_mongo
	go test -race -v ./integration_tests
	go test -race -v ./cs_integration_tests

start_mongo:
	./mongo.sh start

stop_mongo:
	./mongo.sh stop

update_deps:
	go get -t -u ./... && go mod tidy && go mod vendor
