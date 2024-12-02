# Devpod Desktop

[Open Example Devpod](devpod://open?workspace=vscode-remote-try-go&source=https://github.com/Microsoft/vscode-remote-try-go&provider=docker)

## Development

1. Install [NodeJS](https://nodejs.org/en/)
2. Install [Yarn](https://yarnpkg.com/getting-started/install) and make sure you
   use yarn v1, by running `yarn set version 1.22.1`
3. Install [Rust](https://www.rust-lang.org/tools/install)
4. Install [Go](https://go.dev/doc/install)
5. Run `./hack/rebuild.sh` from the root directory of the repo
6. Install dependencies with `yarn` in the `desktop` directory
7. Run `yarn tauri dev` in the `desktop` directory

### Build dependencies

To build the app on Linux, you will need the following dependencies:

```bash
sudo apt-get install libappindicator3-1 libgdk-pixbuf2.0-0 libbsd0 libxdmcp6 \
  libwmf-0.2-7 libwmf-0.2-7-gtk libgtk-3-0 libwmf-dev libwebkit2gtk-4.0-37 \
  librust-openssl-sys-dev librust-glib-sys-dev
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.0-dev \
  libayatana-appindicator3-dev librsvg2-dev file build-essential
```

### Additional Information

Make sure all of your dependencies are installed and up to date by running `yarn`
and `cd src-tauri && cargo update`.

Frontend code lives in `src`
Backend code lives in `src-tauri`

Entrypoint for the application is the `main` function in `src-tauri/main.rs`.
It instructs tauri to set up the application, bootstrap the webview and serve our
static assets. As of now, we just bundle all of the javascript into one file, so
we don't have any prerendering or code splitting going on.

To spin up the application in development mode, run `yarn tauri dev`. It will
report both the frontend webserver output (vite) and the backend logs to your
current terminal.
Tauri should automatically restart the app if your backend code changes and vite
is responsible for hot module updates in the frontend.
Enable debug logging to stdout during development with `DEBUG=true yarn tauri dev`.

If you just want to preview the project locally, make sure to disabled the auto
update feature by setting `desktop/src-tauri/tauri.conf.json->updater.active=false`.
Please be careful not to commit this change later on.
Once you're happy with the current state, give it a spin in release mode by running
`yarn tauri build`. You can find the packaged version of the application in the
`src-tauri/target/release/{PLATFORM}` folder.

## Check Type Errors

Run `yarn types:check` to check for errors

## Versioning

The apps version is determined by the one in `package.json`. Be careful not to add
one in `tauri.conf.json` as it override the current one.
You can upgrade the version manually or run `yarn version ...`

## Build desktop app

If your development environment is setup successfully and you're able to run
`yarn desktop:dev` without problems, you also should be able to build the app
locally by runnning `yarn desktop:build:dev`.
Notice the `:dev` suffix, if you omit this it'll try to build with the updater
enabled. This won't work on your machine as it requires sensitive information.

The output of the command can be found in `desktop/src-tauri/target/release/bundle`.
