#!/bin/bash
set -x
set -eu

PROJECT_NAME="router"

bundle install --path "${HOME}/bundles/${JOB_NAME}"

source ./build_gopath.sh
go build -v -o ${PROJECT_NAME}

go test ./triemux
USE_COMPILED_ROUTER=1 bundle exec rspec
