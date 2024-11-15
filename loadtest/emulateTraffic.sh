#!/bin/zsh

# SSH to the workspace and execute command
for j in $(seq 1 $NUM_COMMANDS_PER_WORKSPACE);
do
    ./generateLoad.sh $1
    sleep 1
done
