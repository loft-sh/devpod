#!/bin/zsh

#kubectl -n devpod-pro set env deployment/loft LOFTDEBUG=true

kubectl -n devpod-pro port-forward $(kubectl -n devpod-pro get pods -l app=loft -o jsonpath="{.items[0].metadata.name}") 8080:8080
