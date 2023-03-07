## Development Setup

1. Clone the repository locally
2. If you want to change something in DevPod agent code:
   1. Exchange the URL in [DefaultAgentDownloadURL](./pkg/agent/agent.go) with a custom public repository release you have created. 
   2. Build devpod via: `./hack/rebuild.sh`
   3. Upload `test/devpod-linux-amd64` and `test/devpod-linux-arm64` to the public repository release assets.
3. Build devpod via: `./hack/rebuild.sh`
4. Configure local provider via `devpod use provider local`
5. Start devpod in vscode with `devpod up examples/simple`
