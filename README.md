<br>
<a href="https://www.devpod.sh">
  <picture width="500">
    <source media="(prefers-color-scheme: dark)" srcset="docs/static/media/devpod_dark.png">
    <img alt="DevPod wordmark" width="500" src="docs/static/media/devpod.png">
  </picture>
</a>

### **[Website](https://www.devpod.sh)** • **[Quickstart](https://www.devpod.sh/docs/getting-started/install)** • **[Documentation](https://www.devpod.sh/docs/what-is-devpod)** • **[Blog](https://loft.sh/blog)** • **[𝕏 (Twitter)](https://x.com/loft_sh)** • **[Slack](https://slack.loft.sh/)**

[![Join us on Slack!](docs/static/media/slack.svg)](https://slack.loft.sh/) [![Open in DevPod!](https://devpod.sh/assets/open-in-devpod.svg)](https://devpod.sh/open#https://github.com/loft-sh/devpod)

**[We are hiring!](https://www.loft.sh/careers) Come build the future of remote development environments with us.**

DevPod is a client-only tool to create reproducible developer environments based on a [devcontainer.json](https://containers.dev/) on any backend. Each developer environment runs in a container and is specified through a [devcontainer.json](https://containers.dev/). Through DevPod providers, these environments can be created on any backend, such as the local computer, a Kubernetes cluster, any reachable remote machine, or in a VM in the cloud.

![Codespaces](docs/static/media/codespaces-but.png)

You can think of DevPod as the glue that connects your local IDE to a machine where you want to develop. So depending on the requirements of your project, you can either create a workspace locally on the computer, on a beefy cloud machine with many GPUs, or a spare remote computer. Within DevPod, every workspace is managed the same way, which also makes it easy to switch between workspaces that might be hosted somewhere else.

![DevPod Flow](docs/static/media/devpod-flow.gif)

## Quickstart

Download DevPod Desktop:
- [MacOS Silicon/ARM](https://github.com/loft-sh/devpod/releases/latest/download/DevPod_macos_aarch64.dmg)
- [MacOS Intel/AMD](https://github.com/loft-sh/devpod/releases/latest/download/DevPod_macos_x64.dmg)
- [Windows](https://github.com/loft-sh/devpod/releases/latest/download/DevPod_windows_x64_en-US.msi)
- [Linux AppImage](https://github.com/loft-sh/devpod/releases/latest/download/DevPod_linux_amd64.AppImage)

Take a look at the [DevPod Docs](https://devpod.sh/docs/getting-started/install) for more information.

## Why DevPod?

DevPod reuses the open [DevContainer standard](https://containers.dev/) (used by GitHub Codespaces and VSCode DevContainers) to create a consistent developer experience no matter what backend you want to use.

Compared to hosted services such as Github Codespaces, JetBrains Spaces, or Google Cloud Workstations, DevPod has the following advantages:
* **Cost savings**: DevPod is usually around 5-10 times cheaper than existing services with comparable feature sets because it uses bare virtual machines in any cloud and shuts down unused virtual machines automatically.
* **No vendor lock-in**: Choose whatever cloud provider suits you best, be it the cheapest one or the most powerful, DevPod supports all cloud providers. If you are tired of using a provider, change it with a single command.
* **Local development**: You get the same developer experience also locally, so you don't need to rely on a cloud provider at all.
* **Cross IDE support**: VSCode and the full JetBrains suite is supported, all others can be connected through simple ssh.
* **Client-only**: No need to install a server backend, DevPod runs only on your computer.
* **Open-Source**: DevPod is 100% open-source and extensible. A provider doesn't exist? Just create your own.
* **Rich feature set**: DevPod already supports prebuilds, auto inactivity shutdown, git & docker credentials sync, and many more features to come.
* **Desktop App**: DevPod comes with an easy-to-use desktop application that abstracts all the complexity away. If you want to build your own integration, DevPod offers a feature-rich CLI as well.
