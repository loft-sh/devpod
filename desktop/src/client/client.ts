import { UseToastOptions } from "@chakra-ui/react"
import { app, event, path } from "@tauri-apps/api"
import { invoke } from "@tauri-apps/api/core"
import { Theme as TauriTheme, getCurrentWindow } from "@tauri-apps/api/window"
import * as clipboard from "@tauri-apps/plugin-clipboard-manager"
import * as dialog from "@tauri-apps/plugin-dialog"
import * as fs from "@tauri-apps/plugin-fs"
import * as log from "@tauri-apps/plugin-log"
import * as os from "@tauri-apps/plugin-os"
import * as process from "@tauri-apps/plugin-process"
import * as shell from "@tauri-apps/plugin-shell"
import { Command } from "@tauri-apps/plugin-shell"
import * as updater from "@tauri-apps/plugin-updater"
import { TSettings } from "../contexts"
import { Release } from "../gen"
import { Result, Return, hasCapability, isError, noop } from "../lib"
import { TCommunityContributions, TProInstance, TUnsubscribeFn } from "../types"
import { Command as DevPodCommand } from "./command"
import { ContextClient } from "./context"
import { IDEsClient } from "./ides"
import { ProClient } from "./pro"
import { DaemonClient } from "./pro/client"
import { ProvidersClient } from "./providers"
import { TAURI_SERVER_URL } from "./tauriClient"
import { WorkspacesClient } from "./workspaces"

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
    | Readonly<{
        type: "OpenProInstance"
        host: string | null
      }>
    | Readonly<{
        type: "LoginRequired"
        host: string
        provider: string
      }>
}
type TChannelName = keyof TChannels
type TClientEventListener<TChannel extends TChannelName> = (payload: TChannels[TChannel]) => void
type TClientSettings = Pick<
  TSettings,
  | "debugFlag"
  | "additionalCliFlags"
  | "dotfilesUrl"
  | "additionalEnvVars"
  | "sshKeyPath"
  | "httpProxyUrl"
  | "httpsProxyUrl"
  | "noProxy"
>
export type TPlatform = Awaited<ReturnType<typeof os.platform>>
export type TArch = Awaited<ReturnType<typeof os.arch>>

class Client {
  public readonly workspaces = new WorkspacesClient()
  public readonly providers = new ProvidersClient()
  public readonly ides = new IDEsClient()
  public readonly context = new ContextClient()
  public readonly pro = new ProClient("")

  public setSetting<TSettingName extends keyof TClientSettings>(
    name: TSettingName,
    value: TSettings[TSettingName]
  ) {
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
    if (name === "dotfilesUrl") {
      this.workspaces.setDotfilesFlag(value as string)
    }
    if (name === "sshKeyPath") {
      this.workspaces.setSSHKeyPath(value as string)
    }
    if (name === "additionalEnvVars") {
      DevPodCommand.ADDITIONAL_ENV_VARS = value as string
    }
    if (name === "httpProxyUrl") {
      DevPodCommand.HTTP_PROXY = value as string
    }
    if (name === "httpsProxyUrl") {
      DevPodCommand.HTTPS_PROXY = value as string
    }
    if (name === "noProxy") {
      DevPodCommand.NO_PROXY = value as string
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

  // emitEvent publishes to a given channel and invokes the corresponding handler.
  // This is only intended to be used for debugging right now.
  public emitEvent<T extends TChannelName>(e: TChannels[T]) {
    event.emit("event", e)
  }

  public fetchPlatform(): TPlatform {
    return os.platform()
  }

  public pathSeparator(): string {
    return path.sep()
  }

  public fetchArch(): TArch {
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
      const res = await fetch(TAURI_SERVER_URL + "/releases")
      if (!res.ok) {
        return Return.Failed(`Fetch releases: ${res.statusText}`)
      }
      const releases = (await res.json()) as readonly Release[]

      return Return.Value(releases)
    } catch (e) {
      // return empty list if error during development
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

  public async getDir(
    dir: Extract<keyof typeof fs.BaseDirectory, "AppData" | "AppLog" | "Home"> | "SSH"
  ): Promise<string> {
    switch (dir) {
      case "AppData": {
        return path.appDataDir()
      }
      case "AppLog": {
        return await path.appLogDir()
      }
      case "Home": {
        return await path.homeDir()
      }
      case "SSH": {
        return await path.join(await path.homeDir(), ".ssh")
      }
    }
  }

  public async openDir(
    dir: Extract<keyof typeof fs.BaseDirectory, "AppData" | "AppLog">
  ): Promise<void> {
    try {
      let p = await this.getDir(dir)
      if (dir === "AppLog") {
        p = await path.join(p, "DevPod.log")
      }

      shell.open(p)
    } catch {
      // noop for now
    }
  }

  public async selectFromDir(title?: string): Promise<string | null> {
    return dialog.open({ title, directory: true, multiple: false })
  }

  public async selectFileYaml(): Promise<string | string[] | null> {
    return dialog.open({
      filters: [{ name: "yaml", extensions: ["yml", "yaml"] }],
      directory: false,
      multiple: false,
    })
  }

  public async selectFile(defaultPath?: string): Promise<string | string[] | null> {
    return dialog.open({ directory: false, multiple: false, defaultPath })
  }

  public async copyFile(src: string, dest: string): Promise<void> {
    return fs.copyFile(src, dest)
  }

  public async copyFilePaths(src: string[], dest: string[]) {
    return this.copyFile(await path.join(...src), await path.join(...dest))
  }

  public async writeTextFile(targetPath: string[], data: string) {
    return fs.writeTextFile(await path.join(...targetPath), data)
  }

  public async readFile(targetPath: string[]) {
    return fs.readFile(await path.join(...targetPath))
  }

  public async readTextFile(targetPath: string[]) {
    return fs.readTextFile(await path.join(...targetPath))
  }

  public async writeFile(targetPath: string[], data: Uint8Array) {
    return fs.writeFile(await path.join(...targetPath), data)
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

      const result = await Command.create("run-path-devpod-cli", ["version"]).execute()
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
      const update = await updater.check()
      if (!update) {
        return Return.Ok()
      }

      await update.install()

      return Return.Ok()
    } catch (e) {
      return Return.Failed(`${e}`)
    }
  }

  public async restart(): Promise<void> {
    await process.relaunch()
  }
  public async closeCurrentWindow(): Promise<void> {
    await getCurrentWindow().close()
  }

  public async getSystemTheme(): Promise<TauriTheme | null> {
    return getCurrentWindow().theme()
  }

  public log(level: "debug" | "info" | "warn" | "error", message: string) {
    const logFn = log[level]
    logFn(message)
  }

  public getProClient(proInstance: TProInstance): ProClient | DaemonClient {
    if (hasCapability(proInstance, "daemon")) {
      return new DaemonClient(proInstance.host!)
    } else {
      return new ProClient(proInstance.host!)
    }
  }
}

// Singleton client
export const client = new Client()
