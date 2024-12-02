# Development

## Development Setup

1. Clone the repository locally
2. If you want to change something in DevPod agent code:
   1. Exchange the URL in [DefaultAgentDownloadURL](./pkg/agent/agent.go) with a
      custom public repository release you have created.
   2. Build devpod via: `./hack/rebuild.sh`
   3. Upload `test/devpod-linux-amd64` and `test/devpod-linux-arm64` to the public
      repository release assets.
3. Build devpod via: `./hack/rebuild.sh` (asking for sudo password)
4. Add docker provider via `devpod provider add docker`
5. Configure docker provider via `devpod use provider docker`
6. Start devpod in vscode with `devpod up examples/simple`

## Build from source

Prerequisites CLI:

- [Go 1.20](https://go.dev/doc/install)

Once installed, run
`CGO_ENABLED=0 go build -ldflags "-s -w" -o devpod-cli`

Prerequisites GUI:

- [NodeJS + yarn](https://nodejs.org/en/)
- [Rust](https://www.rust-lang.org/tools/install)
- [Go](https://go.dev/doc/install)

To build the app on Linux, you will need the following dependencies:

```bash
sudo apt-get install libappindicator3-1 libgdk-pixbuf2.0-0 libbsd0 libxdmcp6 \
  libwmf-0.2-7 libwmf-0.2-7-gtk libgtk-3-0 libwmf-dev libwebkit2gtk-4.0-37 \
  librust-openssl-sys-dev librust-glib-sys-dev
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.0-dev \
  libayatana-appindicator3-dev librsvg2-dev
```

Once installed, run

- `cd desktop`
- `yarn tauri build --config src-tauri/tauri-dev.conf.json`

The application should be in `desktop/src-tauri/target/release`

## Provider

Head over to [the docs](https://devpod.sh/docs/developing-providers/quickstart)
for an introduction into developing your own providers

### Publish your provider

Once you're provider is ready, update

- `community.yaml`
- `docs/pages/managing-providers/add-provider.mdx`

to get your provider featured both in the documentation and the UI

## Deeplinks

DevPod Desktop can handle deep links to perform various actions, like opening or
importing workspaces.
The scheme is:

protocol: `devpod://`
host: `command`
searchParams: `foo=bar&fizz=buzz`

resulting in a full url string of `devpod://command?foo=bar&fizz=buzz`. For more
information, take a look at the indvidual command sections below.

### Open Workspace

Open a workspace based on a workspace source. Similar to `devpod up`, but shareable

host: `open`
searchParams: `source` (required), `workspace`, `provider`, `ide`

`devpod://open?source=your-url-encoded-source&workspace=my-workspace&provider=docker&ide=vscode`

### Import Workspace

Import a remote DevPod.Pro workspace into your local client

host: `import`
searchParams: `workspace_id` (required), `workspace_uid` (required),
`devpod_pro_host` (required), `options`
