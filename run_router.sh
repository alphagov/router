#!/bin/bash

set -eu

source ./build_gopath.sh

go run router.go main.go router_api.go
