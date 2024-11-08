#!/bin/zsh

export NUM_WORKSPACES=10

# Start the workspaces
for i in $(seq 1 $NUM_WORKSPACES);
do
    devpod delete "load$i"
done