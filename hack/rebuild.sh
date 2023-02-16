#!/usr/bin/env bash
sudo go build -o /usr/local/bin/devpod
GOOS=linux GOARCH=amd64 go build -o test/devpod-linux-amd64
GOOS=linux GOARCH=arm64 go build -o test/devpod-linux-arm64
rm -R $TMPDIR/devpod-cache
