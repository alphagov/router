#!/bin/bash
set -x
set -eu

# The repository owner and name
REPO="alphagov/router"
# The name of the binary to output
BINARY=$(basename "$REPO")

# Define the Jenkins workspace root in case we're not running through Jenkins
: ${WORKSPACE:=$PWD}

# Go projects need to be built from with a Go workspace, which is a directory
# tree with a specific format.  The default checkout path isn't enough, so we
# need to create that workspace and build from within there.  The GOPATH
# envionment variable points to the location of the workspace.
#
# See https://golang.org/doc/code.html#Workspaces for more details.
BUILD_DIR="__build"
export GOPATH="$WORKSPACE/$BUILD_DIR"

# Define the location within the GOPATH that the source should reside
SRC_PATH="$GOPATH/src/github.com/$REPO"

# Recreate the GOPATH workspace from scratch
rm -rf "$GOPATH" && mkdir -p "$GOPATH/bin" "$SRC_PATH"

# Copy the whole repo content into the build tree
rsync -a ./ "$SRC_PATH" --exclude="$BUILD_DIR"

# Move in to the source path, then build the binary into the Jenkins workspace
# root (for later packaging) and run the tests
(cd "$SRC_PATH" && BINARY="$WORKSPACE/$BINARY" make clean build test)

# Output the version that was built
"./$BINARY" -version
