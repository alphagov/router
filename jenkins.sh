#!/bin/bash
set -x
set -eu

BUNDLER_ARGS=""
if [ -n "${JOB_NAME-}" ]; then
  # Use persistent path outside workspace when actually running on Jenkins.
  BUNDLER_ARGS="--path ${HOME}/bundles/${JOB_NAME}"
fi
bundle install ${BUNDLER_ARGS}

make
USE_COMPILED_ROUTER=1 make test
