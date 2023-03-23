# Documentation

This website is built using [Docusaurus 2](https://v2.docusaurus.io/), a modern static website generator.

### Installation

```
$ yarn
```

### Local Development

```
$ yarn start
```

This command starts a local development server and open up a browser window. Most changes are reflected live without having to restart the server.

### Build

```
$ yarn build
```

This command generates static content into the `build` directory and can be served using any static contents hosting service.


## Creating New Versions

### 1. Generate Command Docs
```bash
cd ../ # main project directory
go run -mod=vendor ./hack/gen-docs.go
```

### 2. Create Version
```bash
yarn run docusaurus docs:version 0.1
```
