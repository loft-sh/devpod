#!/usr/bin/env bash

set -e

sudo go build -o /usr/local/bin/devpod
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o test/devpod-linux-amd64
#upx "test/devpod-linux-amd64"
GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o test/devpod-linux-arm64
#upx "test/devpod-linux-arm64"
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o test/devpod-darwin-arm64
#upx "test/devpod-darwin-arm64"
rm -R $TMPDIR/devpod-cache
