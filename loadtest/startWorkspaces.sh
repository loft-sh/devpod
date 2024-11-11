#!/bin/zsh

export NUM_WORKSPACES=20

# Start the workspaces
for i in $(seq 1 $NUM_WORKSPACES);
do
    devpod up --id "load$i" --ide none https://github.com/loft-sh/devpod-example-devops >/dev/null
done

wait