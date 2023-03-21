#!/usr/bin/env bash

set -e

BUILD_DIR="${BUILDDIR:=test}"
SRC_DIR="${SRCDIR:=.}"

# Create directory if it doesn't exist
if [ ! -d $BUILD_DIR ]
then
    mkdir ./$BUILD_DIR
fi

GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $BUILD_DIR/devpod-linux-amd64 $SRC_DIR
GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o $BUILD_DIR/devpod-linux-arm64 $SRC_DIR
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o $BUILD_DIR/devpod-darwin-arm64 $SRC_DIR
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $BUILD_DIR/devpod-darwin-amd64 $SRC_DIR
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o $BUILD_DIR/devpod-windows-amd64 $SRC_DIR

sudo go build -o /usr/local/bin/devpod
cp $BUILD_DIR/devpod-linux-amd64 desktop/src-tauri/bin/devpod-x86_64-unknown-linux-gnu
cp $BUILD_DIR/devpod-linux-arm64 desktop/src-tauri/bin/devpod-aarch64-unknown-linux-gnu
cp $BUILD_DIR/devpod-darwin-amd64 desktop/src-tauri/bin/devpod-x86_64-apple-darwin
cp $BUILD_DIR/devpod-darwin-arm64 desktop/src-tauri/bin/devpod-aarch64-apple-darwin


#upx "test/devpod-linux-amd64"
#upx "test/devpod-linux-arm64"
#upx "test/devpod-darwin-arm64"
rm -R $TMPDIR/devpod-cache