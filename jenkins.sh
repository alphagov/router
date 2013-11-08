#!/bin/bash
set -x
set -eu

PROJECT_NAME="router"
source ./build_gopath.sh
go build -v -o ${PROJECT_NAME}
go test ./trie ./triemux

BUNDLER_ARGS=""
if [ -n "${JOB_NAME-}" ]; then
  # Use persistent path outside workspace when actually running on Jenkins.
  BUNDLER_ARGS="--path ${HOME}/bundles/${JOB_NAME}"
fi

bundle install ${BUNDLER_ARGS}
USE_COMPILED_ROUTER=1 bundle exec rspec
