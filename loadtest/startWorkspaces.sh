#!/bin/zsh

export NUM_WORKSPACES=20

# Start the workspaces
for i in $(seq 1 $NUM_WORKSPACES);
do
    devpod up --id "load$i" --ide none https://github.com/kubernetes/kubernetes &
    sleep 10
done

wait
