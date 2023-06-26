#!/usr/bin/env bash

set -e

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o test/devpod-cli-linux-amd64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o test/devpod-cli-linux-arm64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o test/devpod-cli-darwin-arm64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o test/devpod-cli-darwin-amd64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o test/devpod-cli-windows-amd64
go build -o test/devpod && sudo mv test/devpod /usr/local/bin/
cp test/devpod-cli-linux-amd64 test/devpod-linux-amd64
cp test/devpod-cli-linux-arm64 test/devpod-linux-arm64
cp test/devpod-cli-linux-amd64 desktop/src-tauri/bin/devpod-cli-x86_64-unknown-linux-gnu
cp test/devpod-cli-linux-arm64 desktop/src-tauri/bin/devpod-cli-aarch64-unknown-linux-gnu
cp test/devpod-cli-darwin-amd64 desktop/src-tauri/bin/devpod-cli-x86_64-apple-darwin
cp test/devpod-cli-darwin-arm64 desktop/src-tauri/bin/devpod-cli-aarch64-apple-darwin

rm -R $TMPDIR/devpod-cache 2>/dev/null || true
mkdir -p $TMPDIR/devpod-cache
cp test/devpod-cli-linux-amd64 $TMPDIR/devpod-cache/devpod-linux-amd64
cp test/devpod-cli-linux-arm64 $TMPDIR/devpod-cache/devpod-linux-arm64
