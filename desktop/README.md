# Devpod Desktop

## Development

Prerequisites:

- [NodeJS + yarn](https://nodejs.org/en/)
- [Rust](https://www.rust-lang.org/tools/install)
- `devpod` binary for your platform in `./src-tauri/bin/`. Needs to be named according to your platform, see [rustc docs](https://doc.rust-lang.org/nightly/rustc/platform-support.html) for details.
  You can build the devpod binary for your current platform by running `cd .. && go build .`.
  For example for macOS:
  ```sh
  cd ..
  go build -o ./desktop/src-tauri/bin/devpod -ldflags "-s -w"
  cd desktop
  mv src-tauri/bin/devpod src-tauri/bin/devpod-x86_64-apple-darwin
  ```

Make sure all of your dependencies are installed and up to date by running `yarn` and `cd src-tauri && cargo update`.

Frontend code lives in `src`
Backend code lives in `src-tauri`

Entrypoint for the application is the `main` function in `src-tauri/main.rs`. It instructs tauri to set up the application, bootstrap the webview and serve our static assets.
As of now, we just bundle all of the javascript into one file, so we don't have any prerendering or code splitting going on.

To spin up the application in development mode, run `yarn tauri dev`. It will report both the frontend webserver output (vite) and the backend logs to your current terminal.
Tauri should automatically restart the app if your backend code changes and vite is responsible for hot module updates in the frontend.

Once you're happy with the current state, give it a spin in release mode by running `yarn tauri build`. You can find the packaged version of the application in the `src-tauri/target/release/{PLATFORM}` folder.
