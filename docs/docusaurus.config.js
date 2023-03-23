__webpack_public_path__ = "/docs/"

module.exports = {
  title: 'DevPod docs | DevContainers everywhere',
  tagline: 'DevContainers everywhere',
  url: 'https://devpod.sh',
  baseUrl: __webpack_public_path__,
  favicon: '/media/devpod-favicon.png',
  organizationName: 'loft-sh', // Usually your GitHub org/user name.
  projectName: 'devpod', // Usually your repo name.
  themeConfig: {
    colorMode: {
      disableSwitch: true,
    },
    navbar: {
      logo: {
        alt: 'devpod',
        src: '/media/devpod-logo.png',
        href: 'https://devpod.sh/',
        target: '_self',
      },
      items: [
        {
          href: 'https://devpod.sh/',
          label: 'Website',
          position: 'left',
          target: '_self'
        },
        {
          to: '/docs/what-is-devpod',
          label: 'Docs',
          position: 'left'
        },
        {
          href: 'https://loft.sh/blog',
          label: 'Blog',
          position: 'left',
          target: '_self'
        },
        {
          href: 'https://slack.loft.sh/',
          className: 'slack-link',
          'aria-label': 'Slack',
          position: 'right',
        },
        {
          href: 'https://github.com/loft-sh/vcluster',
          className: 'github-link',
          'aria-label': 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'light',
      links: [],
      copyright: `Copyright © ${new Date().getFullYear()} <a href="https://loft.sh/">Loft Labs, Inc.</a>`,
    },
  },
  presets: [
    [
      '@docusaurus/preset-classic',
      {
        docs: {
          path: 'pages',
          routeBasePath: '/',
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl:
            'https://github.com/loft-sh/devpod/edit/main/docs/',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
  ],
  plugins: [],
  scripts: [
    {
      src:
        'https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.0/clipboard.min.js',
      async: true,
    },
    {
      src:
        '/docs/js/custom.js',
      async: true,
    },
  ],
};
