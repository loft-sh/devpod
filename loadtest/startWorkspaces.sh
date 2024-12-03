#!/bin/zsh

export NUM_WORKSPACES=20

# Start the workspaces
for i in $(seq 11 $NUM_WORKSPACES);
do
    devpod up --id "loadtest$i" --debug --ide none http://github.com/kubernetes/kubernetes
done

wait
