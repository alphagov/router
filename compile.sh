#!/bin/bash

set -eu

REPO_ROOT=$(dirname $(readlink -f $0))
cd $REPO_ROOT

export GOPATH="$REPO_ROOT/gopath.tmp"
export GOBIN="$GOPATH/bin"

PROJECT_PATH="github.com/alphagov"
mkdir -p ${GOPATH}/src/${PROJECT_PATH}

rm -f ${GOPATH}/src/${PROJECT_PATH}/${PROJECT_NAME}
ln -s ../../../.. ${GOPATH}/src/${PROJECT_PATH}/${PROJECT_NAME}

go get -v -d
go build -v -o ${PROJECT_NAME}
