#!/bin/bash
set -x
set -eu

bundle install --path "${HOME}/bundles/${JOB_NAME}"

./compile.sh
bundle exec rspec

bundle exec fpm \
  -s dir -t deb \
  -n ${PROJECT_NAME} -v 0.${BUILD_NUMBER} \
  --prefix /usr/bin \
  ${PROJECT_NAME}
