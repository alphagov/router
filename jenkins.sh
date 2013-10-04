#!/bin/bash
set -x
set -eu

export GOPATH="$(pwd)/gopath.tmp"
export GOBIN="$GOPATH/bin"

rm -rf $GOPATH
mkdir $GOPATH

PROJECT_PATH="github.com/alphagov"
PROJECT_NAME="router"
mkdir -p ${GOPATH}/src/${PROJECT_PATH}
ln -s ../../../.. ${GOPATH}/src/${PROJECT_PATH}/${PROJECT_NAME}

go get -v -d
go build -v -o ${PROJECT_NAME}

bundle install --path "${HOME}/bundles/${JOB_NAME}"
bundle exec fpm \
  -s dir -t deb \
  -n ${PROJECT_NAME} -v ${BUILD_NUMBER} \
  --prefix /usr/bin \
  ${PROJECT_NAME}
