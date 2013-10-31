#!/bin/bash

set -eu

REPO_ROOT=$(dirname ${BASH_SOURCE[0]})
cd $REPO_ROOT
REPO_ROOT=$(pwd) # Resolve relative paths

export GOPATH="$REPO_ROOT/gopath.tmp"
export GOBIN="$GOPATH/bin"

PROJECT_PATH="github.com/alphagov/router"
mkdir -p $(dirname ${GOPATH}/src/${PROJECT_PATH})

rm -f ${GOPATH}/src/${PROJECT_PATH}
ln -s ../../../.. ${GOPATH}/src/${PROJECT_PATH}

go get -v -d
