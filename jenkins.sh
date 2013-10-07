#!/bin/bash
set -x
set -eu

PROJECT_NAME="router"

bundle install --path "${HOME}/bundles/${JOB_NAME}"

source ./build_gopath.sh
go build -v -o ${PROJECT_NAME}
USE_COMPILED_ROUTER=1 bundle exec rspec

bundle exec fpm \
  -s dir -t deb \
  -n ${PROJECT_NAME} -v 0.${BUILD_NUMBER} \
  --prefix /usr/bin \
  ${PROJECT_NAME}
