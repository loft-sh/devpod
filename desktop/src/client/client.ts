import {
  app,
  clipboard,
  dialog,
  fs,
  invoke,
  os,
  path,
  process,
  shell,
  event,
  updater,
  window as tauriWindow,
} from "@tauri-apps/api"
import { Command } from "@tauri-apps/api/shell"
import { TSettings } from "../contexts"
import { Result, Return, isError, noop } from "../lib"
import { TCommunityContributions, TUnsubscribeFn } from "../types"
import { ContextClient } from "./context"
import { IDEsClient } from "./ides"
import { ProvidersClient } from "./providers"
import { WorkspacesClient } from "./workspaces"
import { UseToastOptions } from "@chakra-ui/react"
import { Release } from "../gen"
import { ProClient } from "./pro"

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
    | Readonly<{ type: "OpenWorkspaceFailed" }>
    | Readonly<{
        type: "OpenWorkspace"
        workspace_id: string | null
        provider_id: string | null
        ide: string | null
        source: string
      }>
}
type TChannelName = keyof TChannels
type TClientEventListener<TChannel extends TChannelName> = (payload: TChannels[TChannel]) => void
type TClientSettings = Pick<TSettings, "debugFlag">
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
    if (name === "debugFlag") {
      this.workspaces.setDebug(value)
      this.providers.setDebug(value)
      this.ides.setDebug(value)
      this.pro.setDebug(value)
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
      const releases = await invoke<readonly Release[]>("get_releases")

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
  public async isCLIInstalled(): Promise<Result<boolean>> {
    try {
      const result = await new Command("run-path-devpod-cli", ["version"]).execute()
      if (result.code !== 0) {
        return Return.Value(false)
      }

      return Return.Value(true)
    } catch {
      return Return.Value(false)
    }
  }

  public openLink(link: string): void {
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
}

// Singleton client
export const client = new Client()
