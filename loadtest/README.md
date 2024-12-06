## Load Testing DevPod

### Create the workspaces

`./startWorkspaces.sh`

### Run the load test and generate wait times

`./run.sh`

Update NUM_WORKSPACES or NUM_COMMANDS_PER_WORKSPACE to adjust load signature

### Clean up

`./deleteWorkspaces.sh`

### Things to note

`generateLoad.sh` contains the SSH command to generate load, change the command here to adjust how you want to generate traffic

### Get core dump from loft

```
kubectl -n devpod-pro set env deployment/loft LOFTDEBUG=true

kubectl -n devpod-pro port-forward loft-55df4d875f-j9vnd 8080:8080 &

curl -s -v http://localhost:8080/debug/pprof/heap > $(date '+%Y-%m-%d-%H:%M:%S').out
```

### Profile the server every 30 seconds

```
while true; do curl -s -v http://localhost:8080/debug/pprof/heap > $(date '+%Y-%m-%d-%H:%M:%S').out; sleep 30; done
```

