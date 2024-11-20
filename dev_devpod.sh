#! /usr/bin/env zsh

set -e

NS=${1:-"default"}

if [[ ! $PWD == *"/go/src/devpod"* ]]; then
  echo "Please run this script from the /workspace/loft/devpod directory"
  exit 1
fi

# SKIP_INSTALL=true BUILD_PLATFORMS="linux darwin" ./hack/rebuild.sh
#SKIP_INSTALL=true BUILD_PLATFORMS="linux" ./hack/rebuild.sh
CGO_ENABLED=0 go build -ldflags "-s -w" -o devpod-cli
kubectl -n $NS cp --no-preserve=true ./devpod-cli $(kubectl -n $NS get pods -l app=loft -o jsonpath="{.items[0].metadata.name}"):/usr/local/bin/devpod
