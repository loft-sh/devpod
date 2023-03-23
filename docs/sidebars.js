/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

module.exports = {
  adminSidebar: [
    {
      type: 'doc',
      id: 'what-is-devpod',
    },
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: [
        {
          type: 'doc',
          id: 'getting-started/quickstart',
        },
      ],
    },
    {
      type: 'category',
      label: 'Developing in a Workspace',
      items: [
        {
          type: 'doc',
          id: 'developing-in-workspaces/develop-in-a-workspace',
        },
      ],
    },
    {
      type: 'category',
      label: 'Prebuilding Workspaces',
      items: [
        {
          type: 'doc',
          id: 'prebuilding-workspaces/prebuild-a-workspace',
        },
      ],
    },
    {
      type: 'category',
      label: 'Managing your Machines',
      items: [
        {
          type: 'doc',
          id: 'managing-machines/what-are-machines',
        },
      ],
    },
    {
      type: 'category',
      label: 'Managing your Providers',
      items: [
        {
          type: 'doc',
          id: 'managing-providers/what-are-providers',
        },
      ],
    },
    {
      type: 'category',
      label: 'Developing Providers',
      items: [
        {
          type: 'doc',
          id: 'developing-providers/quickstart',
        },
      ],
    },
    {
      type: 'link',
      label: 'Originally created by Loft',
      href: 'https://loft.sh/',
    },
  ],
};
