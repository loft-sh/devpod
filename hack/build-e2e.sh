#!/usr/bin/env bash

set -e

BUILD_DIR="${BUILDDIR:=test}"
SRC_DIR="${SRCDIR:=.}"

# Create directory if it doesn't exist
if [ ! -d $BUILD_DIR ]
then
    mkdir ./$BUILD_DIR
fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $BUILD_DIR/devpod-linux-amd64 $SRC_DIR
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o $BUILD_DIR/devpod-linux-arm64 $SRC_DIR
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o $BUILD_DIR/devpod-darwin-arm64 $SRC_DIR
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $BUILD_DIR/devpod-darwin-amd64 $SRC_DIR

chmod +x $BUILD_DIR/devpod-linux-amd64
chmod +x $BUILD_DIR/devpod-linux-arm64
chmod +x $BUILD_DIR/devpod-darwin-arm64
chmod +x $BUILD_DIR/devpod-darwin-amd64
mkdir -p /tmp/devpod-cache
cp $BUILD_DIR/devpod-linux-amd64 /tmp/devpod-cache/devpod-linux-amd64
cp $BUILD_DIR/devpod-linux-arm64 /tmp/devpod-cache/devpod-linux-arm64
