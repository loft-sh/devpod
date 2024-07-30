#!/usr/bin/env bash

set -e

if [[ -z "${BUILD_PLATFORMS}" ]]; then
    BUILD_PLATFORMS="linux windows darwin"
fi

for os in $BUILD_PLATFORMS; do
    # FIXME: for arch in amd64 arm64; do
    for arch in arm64; do
        # don't build for arm on windows
        if [[ "$os" == "windows" && "$arch" == "arm64" ]]; then
            continue
        fi
        echo "[INFO] Building for $os/$arch"
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -ldflags "-s -w" -o test/devpod-cli-$os-$arch
    done
done

echo "[INFO] Built binaries for all platforms in test/ directory"
if [[ -z "${SKIP_INSTALL}" ]]; then
    go build -o test/devpod && sudo mv test/devpod /usr/local/bin/
fi

echo "[INFO] Built devpod binary and moved to /usr/local/bin"
if [[ $BUILD_PLATFORMS == *"linux"* ]]; then
    cp test/devpod-cli-linux-amd64 test/devpod-linux-amd64 
    cp test/devpod-cli-linux-arm64 test/devpod-linux-arm64
    cp test/devpod-cli-linux-amd64 desktop/src-tauri/bin/devpod-cli-x86_64-unknown-linux-gnu
    cp test/devpod-cli-linux-arm64 desktop/src-tauri/bin/devpod-cli-aarch64-unknown-linux-gnu
fi
# only copy if darwin is in BUILD_PLATFORMS
if [[ $BUILD_PLATFORMS == *"darwin"* ]]; then
    cp test/devpod-cli-darwin-amd64 desktop/src-tauri/bin/devpod-cli-x86_64-apple-darwin
    cp test/devpod-cli-darwin-arm64 desktop/src-tauri/bin/devpod-cli-aarch64-apple-darwin
fi
echo "[INFO] Copied binaries to desktop/src-tauri/bin"

if [[ $BUILD_PLATFORMS == *"linux"* ]]; then
    rm -R $TMPDIR/devpod-cache 2>/dev/null || true
    mkdir -p $TMPDIR/devpod-cache
    cp test/devpod-cli-linux-amd64 $TMPDIR/devpod-cache/devpod-linux-amd64
    cp test/devpod-cli-linux-arm64 $TMPDIR/devpod-cache/devpod-linux-arm64
    echo "[INFO] Copied binaries to $TMPDIR/devpod-cache"
fi
