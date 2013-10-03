#!/bin/bash
set -x
set -eu

export GOPATH="$(pwd)/gopath.tmp"
export GOBIN="$GOPATH/bin"

rm -rf $GOPATH
mkdir $GOPATH

go get -v
go build -v -o router

bundle install
bundle exec fpm \
  -s dir -t deb \
  -n router -v ${VERSION} \
  --prefix /usr/bin \
  router
