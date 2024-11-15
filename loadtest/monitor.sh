#!/bin/zsh

while true; do curl -s -v http://localhost:8080/debug/pprof/heap > $(date '+%Y-%m-%d-%H:%M:%S').out; sleep 10; done