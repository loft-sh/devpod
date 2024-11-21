#!/bin/zsh

kubectl exec -n devpod $(kubectl -n devpod get pods -l devpod.sh/load-id=$1 -o jsonpath="{.items[0].metadata.name}") -- bash -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 1000000; echo" > /dev/null
