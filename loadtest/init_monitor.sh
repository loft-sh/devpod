#!/bin/zsh

kubectl -n devpod-pro set env deployment/loft LOFTDEBUG=true

kubectl -n devpod-pro port-forward loft-55df4d875f-gd5tw 8080:8080