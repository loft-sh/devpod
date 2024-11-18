#!/bin/zsh

export NUM_WORKSPACES=20

# Start the workspaces
for i in $(seq 1 $NUM_WORKSPACES);
do
    devpod delete --force "loadtest$i" &
    sleep 2
done

wait
