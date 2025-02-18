/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

module.exports = {
  adminSidebar: [
    {
      type: "doc",
      id: "what-is-devpod",
    },
    {
      type: "category",
      label: "Getting Started",
      items: [
        {
          type: "doc",
          id: "getting-started/install",
        },
        {
          type: "doc",
          id: "getting-started/update",
        },
        {
          type: "category",
          label: "Quick Start",
          items: [
            {
              type: "doc",
              id: "quickstart/browser",
            },
            {
              type: "doc",
              id: "quickstart/vscode",
            },
            {
              type: "doc",
              id: "quickstart/jetbrains",
            },
            {
              type: "doc",
              id: "quickstart/ssh",
            },
            {
              type: "doc",
              id: "quickstart/vim",
            },
            {
              type: "doc",
              id: "quickstart/devpod-cli",
            },
          ],
        },
      ],
    },
    {
      type: "category",
      label: "Developing in a Workspace",
      items: [
        {
          type: "doc",
          id: "developing-in-workspaces/what-are-workspaces",
        },
        {
          type: "doc",
          id: "developing-in-workspaces/create-a-workspace",
        },
        {
          type: "doc",
          id: "developing-in-workspaces/connect-to-a-workspace",
        },
        {
          type: "doc",
          id: "developing-in-workspaces/devcontainer-json",
        },
        {
          type: "doc",
          id: "developing-in-workspaces/environment-variables-in-devcontainer-json",
        },
        {
          type: "doc",
          id: "developing-in-workspaces/prebuild-a-workspace",
        },
        {
          type: "doc",
          id: "developing-in-workspaces/dotfiles-in-a-workspace",
        },
        {
          type: "doc",
          id: "developing-in-workspaces/credentials",
        },
        {
          type: "doc",
          id: "developing-in-workspaces/inactivity-timeout",
        },
        {
          type: "doc",
          id: "developing-in-workspaces/stop-a-workspace",
        },
        {
          type: "doc",
          id: "developing-in-workspaces/delete-a-workspace",
        },
      ],
    },
    {
      type: "category",
      label: "Managing your Machines",
      items: [
        {
          type: "doc",
          id: "managing-machines/what-are-machines",
        },
        {
          type: "doc",
          id: "managing-machines/manage-machines",
        },
      ],
    },
    {
      type: "category",
      label: "Managing your Providers",
      items: [
        {
          type: "doc",
          id: "managing-providers/what-are-providers",
        },
        {
          type: "doc",
          id: "managing-providers/add-provider",
        },
        {
          type: "doc",
          id: "managing-providers/update-provider",
        },
        {
          type: "doc",
          id: "managing-providers/delete-provider",
        },
      ],
    },
    {
      type: "category",
      label: "Architecture",
      items: [
        {
          type: "doc",
          id: "how-it-works/overview",
        },
        {
          type: "doc",
          id: "how-it-works/deploy-machines",
        },
        {
          type: "doc",
          id: "how-it-works/deploy-k8s",
        },
        {
          type: "doc",
          id: "how-it-works/building-workspaces",
        },
        {
          type: "doc",
          id: "how-it-works/deploying-workspaces",
        },
      ],
    },
    {
      type: "category",
      label: "Tutorials",
      items: [
        {
          type: "doc",
          id: "tutorials/minikube-vscode-browser",
        },
        {
          type: "doc",
          id: "tutorials/reduce-build-times-with-cache",
        },
        {
          type: "doc",
          id: "tutorials/docker-provider-via-wsl",
        },
      ],
    },
    {
      type: "category",
      label: "Developing Providers",
      items: [
        {
          type: "doc",
          id: "developing-providers/quickstart",
        },
        {
          type: "doc",
          id: "developing-providers/options",
        },
        {
          type: "doc",
          id: "developing-providers/binaries",
        },
        {
          type: "doc",
          id: "developing-providers/agent",
        },
        {
          type: "doc",
          id: "developing-providers/driver",
        },
      ],
    },
    {
      type: "category",
      label: "Troubleshooting",
      items: [
        {
          type: "doc",
          id: "troubleshooting/troubleshooting",
        },
        {
          type: "doc",
          id: "troubleshooting/linux-troubleshooting",
        },
        {
          type: "doc",
          id: "troubleshooting/windows-troubleshooting",
        },
        {
          type: "doc",
          id: "troubleshooting/ide-troubleshooting",
        },
      ],
    },
    {
      type: "category",
      label: "Other topics",
      items: [
        {
          type: "doc",
          id: "other-topics/telemetry",
        },
        {
          type: "doc",
          id: "other-topics/mobile-support",
        }
      ],
    },
    {
      type: "doc",
      id: "licenses/devpod",
    },
    {
      type: "link",
      label: "Open Sourced by Loft",
      href: "https://loft.sh/",
    },
  ],
};
