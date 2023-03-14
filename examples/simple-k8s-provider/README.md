## Simple Kubernetes Provider

This provider starts a privileged Docker-in-Docker pod in a Kubernetes cluster and connects the workspace to it. To use this provider simply clone the repo and run:
```
devpod provider add ./examples/provider.yaml
```

Then start a new workspace via:
```
devpod up github.com/microsoft/vscode-course-sample
```
