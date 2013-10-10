#!/bin/bash
set -eu

[ -x .venv/bin/pip ] || virtualenv .venv
. .venv/bin/activate

pip install -q ghtools

REPO="alphagov/router"
gh-status "$REPO" "$GIT_COMMIT" pending -d "\"Build #${BUILD_NUMBER} is running on Jenkins\"" -u "$BUILD_URL" >/dev/null

if ./jenkins.sh; then
  gh-status "$REPO" "$GIT_COMMIT" success -d "\"Build #${BUILD_NUMBER} succeeded on Jenkins\"" -u "$BUILD_URL" >/dev/null
  exit 0
else
  gh-status "$REPO" "$GIT_COMMIT" failure -d "\"Build #${BUILD_NUMBER} failed on Jenkins\"" -u "$BUILD_URL" >/dev/null
  exit 1
fi
