#!/bin/zsh

mkdir results

export INTERVAL_SECONDS=30

echo "Monitoring heap, go routines, and threads every $INTERVAL_SECONDS seconds ..."

while true; do curl -s -k https://localhost:8080/debug/pprof/heap > ./results/$(date '+%Y-%m-%d-%H:%M:%S').heap; sleep $(echo $INTERVAL_SECONDS); done &

while true; do curl -s -k https://localhost:8080/debug/pprof/goroutine > ./results/$(date '+%Y-%m-%d-%H:%M:%S').cpu; sleep $(echo $INTERVAL_SECONDS); done &

while true; do curl -s -k https://localhost:8080/debug/pprof/threadcreate > ./results/$(date '+%Y-%m-%d-%H:%M:%S').threads; sleep $(echo $INTERVAL_SECONDS); done &

trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT
wait
