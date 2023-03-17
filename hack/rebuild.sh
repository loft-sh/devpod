#!/usr/bin/env bash

set -e

GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o test/devpod-linux-amd64
GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o test/devpod-linux-arm64
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o test/devpod-darwin-arm64
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o test/devpod-darwin-amd64
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o test/devpod-windows-amd64
sudo go build -o /usr/local/bin/devpod
cp test/devpod-linux-amd64 desktop/src-tauri/bin/devpod-x86_64-unknown-linux-gnu
cp test/devpod-linux-arm64 desktop/src-tauri/bin/devpod-aarch64-unknown-linux-gnu
cp test/devpod-darwin-amd64 desktop/src-tauri/bin/devpod-x86_64-apple-darwin
cp test/devpod-darwin-arm64 desktop/src-tauri/bin/devpod-aarch64-apple-darwin


#upx "test/devpod-linux-amd64"
#upx "test/devpod-linux-arm64"
#upx "test/devpod-darwin-arm64"
rm -R $TMPDIR/devpod-cache
