#!/bin/zsh

export NUM_WORKSPACES=20

# Start the workspaces
for i in $(seq 1 $NUM_WORKSPACES);
do
    devpod up --id "loadtest$i" --ide none http://github.com/loft-sh/devpod-example-go
done

wait
