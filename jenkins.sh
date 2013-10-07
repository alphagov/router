#!/bin/bash
set -x
set -eu

export PROJECT_NAME="router"

bundle install --path "${HOME}/bundles/${JOB_NAME}"

./compile.sh
USE_COMPILED_ROUTER=1 bundle exec rspec

bundle exec fpm \
  -s dir -t deb \
  -n ${PROJECT_NAME} -v 0.${BUILD_NUMBER} \
  --prefix /usr/bin \
  ${PROJECT_NAME}
