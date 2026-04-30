.PHONY: all clean build test lint unit_tests integration_tests update_deps
.NOTPARALLEL:

TARGET_MODULE := router
GO_BUILD_ENV := CGO_ENABLED=0
SHELL := /bin/dash

all: build

clean:
	rm -f $(TARGET_MODULE)
	rm -f coverage/*/covmeta.* coverage/*/covcounters.* coverage/report/*.*

build:
	env $(GO_BUILD_ENV) go build -cover -covermode atomic
	GOCOVERDIR=coverage/version ./$(TARGET_MODULE) -version

test: lint unit_tests integration_tests

lint:
	@if ! command -v golangci-lint; then \
		echo "linting uses golangci-lint: you can install it with:\n"; \
		echo "    brew install golangci-lint\n"; \
		exit 1; \
	fi
	golangci-lint run

unit_tests:
	# Using covermode atomic so it can be merged with the integration tests
	go test -cover -covermode atomic -race $$(go list ./... | grep -v integration_tests) -args -test.gocoverdir="${PWD}/coverage/unit"

integration_tests: build
	go test -race -v ./integration_tests

update_deps:
	go get -t -u ./... && go mod tidy && go mod vendor

coverage_report:
	go tool covdata merge -i coverage/unit,coverage/integration/,coverage/version -o coverage/merged/
	go tool covdata textfmt -i coverage/merged/ -o coverage/report/textfmt.txt
	go tool cover -html coverage/report/textfmt.txt -o coverage/report/coverage.html
	go tool cover -func=coverage/report/textfmt.txt | tail -n 1 > coverage/report/overall-coverage.txt
	./coverage/generate-markdown-summary.sh
