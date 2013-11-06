#!/bin/bash

set -eu

REPO_ROOT=$(dirname ${BASH_SOURCE[0]})
cd $REPO_ROOT
REPO_ROOT=$(pwd) # Resolve relative paths

GOPATH_TMP="${REPO_ROOT}/gopath.tmp"
export GOPATH="${GOPATH_TMP}:${REPO_ROOT}/vendor"
export GOBIN="${GOPATH_TMP}/bin"

PROJECT_PATH="github.com/alphagov/router"
mkdir -p $(dirname ${GOPATH_TMP}/src/${PROJECT_PATH})

rm -f ${GOPATH_TMP}/src/${PROJECT_PATH}
ln -s ../../../.. ${GOPATH_TMP}/src/${PROJECT_PATH}

go get -v -d
