import { UseToastOptions } from "@chakra-ui/react"
import {
  app,
  clipboard,
  dialog,
  event,
  fs,
  invoke,
  os,
  path,
  process,
  shell,
  window as tauriWindow,
  updater,
} from "@tauri-apps/api"
import { Command } from "@tauri-apps/api/shell"
import { Theme as TauriTheme } from "@tauri-apps/api/window"
import { TSettings } from "../contexts"
import { Release } from "../gen"
import { Result, Return, isError, noop } from "../lib"
import { TCommunityContributions, TUnsubscribeFn } from "../types"
import { ContextClient } from "./context"
import { IDEsClient } from "./ides"
import { ProClient } from "./pro"
import { ProvidersClient } from "./providers"
import { WorkspacesClient } from "./workspaces"
import { Command as DevPodCommand } from "./command"

// These types have to match the rust types! Make sure to update them as well!
type TChannels = {
  event:
    | Readonly<{
        type: "ShowToast"
        message: string
        title: string
        status: NonNullable<UseToastOptions["status"]>
      }>
    | Readonly<{ type: "ShowDashboard" }>
    | Readonly<{ type: "CommandFailed" }>
    | Readonly<{
        type: "OpenWorkspace"
        workspace_id: string | null
        provider_id: string | null
        ide: string | null
        source: string
      }>
    | Readonly<{
        type: "ImportWorkspace"
        workspace_id: string
        workspace_uid: string
        devpod_pro_host: string
        project: string
        options: Record<string, string> | null
      }>
    | Readonly<{
        type: "SetupPro"
        host: string
        accessKey: string | null
        options: Record<string, string> | null
      }>
}
type TChannelName = keyof TChannels
type TClientEventListener<TChannel extends TChannelName> = (payload: TChannels[TChannel]) => void
type TClientSettings = Pick<
  TSettings,
  "debugFlag" | "additionalCliFlags" | "dotfilesURL" | "additionalEnvVars"
>
export type TPlatform = Awaited<ReturnType<typeof os.platform>>
export type TArch = Awaited<ReturnType<typeof os.arch>>

class Client {
  public readonly workspaces = new WorkspacesClient()
  public readonly providers = new ProvidersClient()
  public readonly ides = new IDEsClient()
  public readonly context = new ContextClient()
  public readonly pro = new ProClient()

  public setSetting<TSettingName extends keyof TClientSettings>(
    name: TSettingName,
    value: TSettings[TSettingName]
  ) {
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    if (name === "debugFlag") {
      const debug: boolean = value as boolean
      this.workspaces.setDebug(debug)
      this.providers.setDebug(debug)
      this.ides.setDebug(debug)
      this.pro.setDebug(debug)
    }
    if (name === "additionalCliFlags") {
      this.workspaces.setAdditionalFlags(value as string)
    }
    if (name === "dotfilesURL") {
      this.workspaces.setDotfilesFlag(value as string)
    }
    if (name === "additionalEnvVars") {
      DevPodCommand.ADDITIONAL_ENV_VARS = value as string
    }
  }
  public ready(): Promise<void> {
    return invoke("ui_ready")
  }

  public async subscribe<T extends TChannelName>(
    channel: T,
    listener: TClientEventListener<T>
  ): Promise<TUnsubscribeFn> {
    // `TClient` is strictly typed so we're fine casting the response as `any`.
    try {
      const unsubscribe = await event.listen<any>(channel, (event) => {
        listener(event.payload)
      })

      return unsubscribe
    } catch {
      return noop
    }
  }

  public fetchPlatform(): Promise<TPlatform> {
    return os.platform()
  }

  public fetchArch(): Promise<TArch> {
    return os.arch()
  }

  public fetchVersion(): Promise<string> {
    return app.getVersion()
  }

  public async fetchCommunityContributions(): Promise<Result<TCommunityContributions>> {
    try {
      const contributions = await invoke<TCommunityContributions>("get_contributions")

      return Return.Value(contributions)
    } catch (e) {
      if (isError(e)) {
        return Return.Failed(e.message)
      }

      const errMsg = "Unable to fetch community contributions"
      if (typeof e === "string") {
        return Return.Failed(`${errMsg}: ${e}`)
      }

      return Return.Failed(errMsg)
    }
  }

  public async fetchReleases(): Promise<Result<readonly Release[]>> {
    try {
      // WARN: This is a workaround for a memory leak in tauri, see https://github.com/tauri-apps/tauri/issues/4026 for more details.
      // tl;dr tauri doesn't release the memory in it's invoke api properly which is specially noticeable with larger payload, like the releases.
      const res = await fetch("http://localhost:25842/releases")
      if (!res.ok) {
        return Return.Failed(`Fetch releases: ${res.statusText}`)
      }
      const releases = (await res.json()) as readonly Release[]

      return Return.Value(releases)
    } catch (e) {
      if (isError(e)) {
        return Return.Failed(e.message)
      }

      const errMsg = "Unable to fetch releases"
      if (typeof e === "string") {
        return Return.Failed(`${errMsg}: ${e}`)
      }

      return Return.Failed(errMsg)
    }
  }

  public async openDir(dir: Extract<keyof typeof fs.BaseDirectory, "AppData">): Promise<void> {
    try {
      let p: string
      switch (dir) {
        case "AppData": {
          p = await path.appDataDir()
          break
        }
      }
      shell.open(p)
    } catch (e) {
      // noop for now
    }
  }

  public async selectFromDir(): Promise<string | string[] | null> {
    return dialog.open({ directory: true, multiple: false })
  }

  public async selectFromFileYaml(): Promise<string | string[] | null> {
    return dialog.open({
      filters: [{ name: "yaml", extensions: ["yml", "yaml"] }],
      directory: false,
      multiple: false,
    })
  }

  public async selectFromFile(): Promise<string | string[] | null> {
    return dialog.open({ directory: false, multiple: false })
  }

  public async copyFile(src: string, dest: string): Promise<void> {
    return fs.copyFile(src, dest)
  }

  public async installCLI(force: boolean = false): Promise<Result<void>> {
    try {
      await invoke("install_cli", { force })

      return Return.Ok()
    } catch (e) {
      if (isError(e)) {
        return Return.Failed(e.message)
      }

      if (typeof e === "string") {
        return Return.Failed(`Failed to install CLI: ${e}`)
      }

      return Return.Failed("Unable to install CLI")
    }
  }

  public async getEnv(name: string): Promise<boolean> {
    return invoke<boolean>("get_env", { name })
  }

  public async isCLIInstalled(): Promise<Result<boolean>> {
    try {
      // we're in a flatpak, we need to check in other paths.
      if (import.meta.env.TAURI_IS_FLATPAK === "true") {
        const home_dir = await this.getEnv("HOME")
        // this will throw if doesn't exist
        const exists = await invoke<boolean>("file_exists", {
          filepath: home_dir + "/.local/bin/devpod",
        })

        return Return.Value(exists)
      }

      const result = await new Command("run-path-devpod-cli", ["version"]).execute()
      if (result.code !== 0) {
        return Return.Value(false)
      }

      return Return.Value(true)
    } catch {
      return Return.Value(false)
    }
  }

  public open(link: string): void {
    shell.open(link)
  }

  public async quit(): Promise<Result<void>> {
    try {
      await process.exit(0)

      return Return.Ok()
    } catch {
      return Return.Failed("Unable to quit")
    }
  }

  public async writeToClipboard(data: string): Promise<Result<void>> {
    try {
      await clipboard.writeText(data)

      return Return.Ok()
    } catch (e) {
      return Return.Failed(`Unable to write to clipboard: ${e}`)
    }
  }

  public async checkUpdates(): Promise<Result<boolean>> {
    try {
      const isOk = await invoke<boolean>("check_updates")

      return Return.Value(isOk)
    } catch (e) {
      return Return.Failed(`${e}`)
    }
  }

  public async fetchPendingUpdate(): Promise<Result<Release>> {
    try {
      const release = await invoke<Release>("get_pending_update")

      return Return.Value(release)
    } catch (e) {
      return Return.Failed(`${e}`)
    }
  }

  public async installUpdate(): Promise<Result<void>> {
    try {
      let unsubscribe: TUnsubscribeFn | undefined
      // Synchronize promise state with update operation
      await new Promise((res, rej) => {
        updater
          .onUpdaterEvent((event) => {
            if (event.status === "ERROR") {
              unsubscribe?.()
              rej(event.error)

              return
            }

            if (event.status === "DONE") {
              unsubscribe?.()
              res(undefined)

              return
            }
          })
          .then(async (u) => {
            unsubscribe = u
            await updater.installUpdate()
          })
      })

      return Return.Ok()
    } catch (e) {
      return Return.Failed(`${e}`)
    }
  }

  public async restart(): Promise<void> {
    await process.relaunch()
  }
  public async closeCurrentWindow(): Promise<void> {
    await tauriWindow.getCurrent().close()
  }

  public async getSystemTheme(): Promise<TauriTheme | null> {
    return tauriWindow.appWindow.theme()
  }
}

// Singleton client
export const client = new Client()
