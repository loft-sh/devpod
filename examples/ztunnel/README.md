# Ztunnel example

```bash
curl -L -o devpod "https://github.com/loft-sh/devpod/releases/latest/download/devpod-darwin-arm64" && sudo install -c -m 0755 devpod /usr/local/bin && rm -f devpod
```

Update the provider

```bash
devpod provider update kubernetes kubernetes
```

Version deployed and version of source code matters. Make sure they match. Pull the latest version of the source code. And copy the devcontainer.json file to the source code. Or you can use the latest version of `gcr.io/istio-testing/build-tools` image.

Make sure you hve a fresh installation of Istio:

```bash
istioctl uninstall --purge -y

istioctl install -y --set profile=ambient --set meshConfig.accessLogFile=/dev/stdout
```

Also, if you want to start from scratch, better you delete the devpod:

```bash
devpod delete . --force
```

Let's deploy an app to test ambient:

```bash
kubectl create ns my-ambient
kubectl label namespace my-ambient istio.io/dataplane-mode=ambient --overwrite
kubectl apply -f sleep.yaml -n my-ambient
kubectl apply -f helloworld.yaml -n my-ambient
```

Verify that app was included to Ambient mode:

```bash
kubectl -n istio-system logs -l k8s-app=istio-cni-node
```

Send traffic to the app:

```bash
kubectl -n my-ambient exec deploy/sleep -- sh -c 'for i in $(seq 1 100); do curl -s -I http://helloworld:5000/hello; done'
```

Output:

```text
HTTP/1.1 200 OK
Server: gunicorn
Date: Tue, 23 Jul 2024 14:21:03 GMT
Connection: keep-alive
Content-Type: text/html; charset=utf-8
Content-Length: 60
```

Verify that the logs are being written to the stdout:

```bash
kubectl -n istio-system logs -l app=ztunnel
```

Output:

```text
2024-07-23T14:21:03.450051Z	info	access	connection complete	src.addr=10.12.0.8:37522 src.workload=sleep-bc9998558-bhv5z src.namespace=my-ambient src.identity="spiffe://cluster.local/ns/my-ambient/sa/sleep" dst.addr=10.12.0.9:15008 dst.hbone_addr=10.12.0.9:5000 dst.service=helloworld.my-ambient.svc.cluster.local dst.workload=helloworld-v1-77489ccb5f-pjbq5 dst.namespace=my-ambient dst.identity="spiffe://cluster.local/ns/my-ambient/sa/default" direction="outbound" bytes_sent=84 bytes_recv=158 duration="118ms"
```

At this point, traffic flows through the ztunnel.

Let's label with `devpod-ztunnel=enabled` only one node to deploy devpod-ztunnel in there:

```shell
FIRST_NODE=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')
kubectl label node $FIRST_NODE devpod-ztunnel=enabled
```

Next command add nodeAffinity to make sure that the upstream ztunnel is not deployed in any node so our test if focused on one node and one devpod-ztunnel:

```bash
kubectl patch daemonset -n istio-system ztunnel --type=merge -p='{"spec":{"template":{"spec":{"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"upstream-ztunnel","operator":"In","values":["no"]}]}]}}}}}}}'
```

You should see that the ztunnel is not deployed anymore. 

**NOTE**: To revert the previous command, run the following:

```bash
# RUN THIS ONLY TO REVERT THE PREVIOUS COMMAND
# kubectl patch daemonset -n istio-system ztunnel --type=merge -p='{"spec":{"template":{"spec":{"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"upstream-ztunnel","operator":"NotIn","values":["no"]}]}]}}}}}}}'
```

Make sure you have docker installed:

```bash
docker --version
```

Go to the root of the project and run the devpod:

**Note** Normally, to start a Devpod is a straight forward simple command. However, given the complexty of ztunnel setup, we need to adjut it a bit.

- `devcontainer.json` is overriden. In our version, we create a `postStartCommand` required to set the secret in the place that ztunnel app can find. Also, some `remoteEnv` are set to help on builing the app in the remote container.
- Using Kind cluster, the `STORAGE_CLASS` is `standard`. If you are using a different cluster, you may need to change it (i.e in EKS it would be `gp2`)
- The template of the pod is also overriden to match what ztunnel needs to run.

```bash
devpod up . --provider-option STORAGE_CLASS=gp2 --provider-option KUBECTL_PATH=/usr/local/bin/kubectl --provider-option KUBERNETES_NAMESPACE=istio-system --provider-option POD_MANIFEST_TEMPLATE=$(pwd)/devpod/pod_manifest.yaml --devcontainer-path devpod/devcontainer.json --ide vscode --debug \
  --recreate --reset
```

You will see DevPod cli copying your project files to the remote container. In the case of ztunnel project, make sure that the `out` folder is deleted before starting devpod. That folder is usully too heavy and unnecesary to be copied to the container.

At the moment, changes in the project when working in the container are not reflected in the local files. To do so, you can run the following command:

```bash
rsync -rlptzv --progress --delete --exclude=.git --exclude=out "ztunnel.devpod:/workspaces/ztunnel" .
```

When the process finishes, you can build the project:

```bash
cargo clean


RUST_LOG="debug" CARGO_TARGET_X86_64_UNKNOWN_LINUX_GNU_RUNNER="sudo -E" cargo build --bin=ztunnel --package=ztunnel --message-format=json
```







