#! /usr/bin/env zsh

set -e

NS=${1:-"default"}

if [[ ! $PWD == *"/go/src/devpod"* ]]; then
  echo "Please run this script from the /workspace/loft/devpod directory"
  exit 1
fi

CGO_ENABLED=1 go build -ldflags "-s -w" -tags profile -race -o devpod-cli
kubectl -n $NS cp --no-preserve=true ./devpod-cli $(kubectl -n $NS get pods -l app=loft -o jsonpath="{.items[0].metadata.name}"):/usr/local/bin/devpod
