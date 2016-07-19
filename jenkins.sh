#!/bin/bash
set -x
set -eu

REPO=alphagov/router
export GOPATH=$PWD/gopath
GO_GITHUB_PATH=$GOPATH/src/github.com
BUILD_PATH=$GO_GITHUB_PATH/$REPO

rm -rf $GOPATH && mkdir -p $GOPATH/bin $BUILD_PATH

rsync -a ./ $BUILD_PATH --exclude=gopath

cd $BUILD_PATH && make
cp ./router $WORKSPACE/router

make test
./router -version
