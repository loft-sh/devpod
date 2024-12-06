#!/bin/zsh

export NUM_WORKSPACES=10
export NUM_COMMANDS_PER_WORKSPACE=1

echo "Running $NUM_WORKSPACES workspaces with $NUM_COMMANDS_PER_WORKSPACE commands each ..."

# SSH to the workspace and execute command
for j in $(seq 1 $NUM_WORKSPACES);
do
    time ./emulateTraffic.sh $j &
    sleep 2
done

# Keep the session active to allow the commands to execute and use STDOUT
wait
