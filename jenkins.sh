#!/bin/bash
set -x
set -eu

make
make test
