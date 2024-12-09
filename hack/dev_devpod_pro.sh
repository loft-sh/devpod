#! /usr/bin/env zsh

set -e

NS=${1:-"default"}
RACE=${2:-"no"}

if [[ ! $PWD == *"/go/src/devpod"* ]]; then
  echo "Please run this script from the /workspace/loft/devpod directory"
  exit 1
fi

if [[ $RACE == "yes" ]]; then
  echo "Building devpod with race detector"
  CGO_ENABLED=1 go build -ldflags "-s -w" -tags profile -race -o devpod-cli
else
  CGO_ENABLED=0 go build -ldflags "-s -w" -tags profile -o devpod-cli
fi

kubectl -n $NS cp --no-preserve=true ./devpod-cli $(kubectl -n $NS get pods -l app=loft -o jsonpath="{.items[0].metadata.name}"):/usr/local/bin/devpod
