import { fs, os, path, shell } from "@tauri-apps/api"
import { listen } from "@tauri-apps/api/event"
import { TSettings } from "../contexts"
import { Result, Return } from "../lib"
import { TUnsubscribeFn } from "../types"
import { ProvidersClient } from "./providers"
import { WorkspacesClient } from "./workspaces"

type TChannels = {}
type TChannelName = keyof TChannels
type TClientEventListener<TChannel extends TChannelName> = (payload: TChannels[TChannel]) => void
type TClientSettings = Pick<TSettings, "debugFlag">
export type TPlatform = Awaited<ReturnType<typeof os.platform>>
export type TArch = Awaited<ReturnType<typeof os.arch>>

class Client {
  public readonly workspaces = new WorkspacesClient()
  public readonly providers = new ProvidersClient()

  public setSetting<TSettingName extends keyof TClientSettings>(
    name: TSettingName,
    value: TSettings[TSettingName]
  ) {
    if (name === "debugFlag") {
      this.workspaces.setDebug(value)
      this.providers.setDebug(value)
    }
  }

  public async subscribe<T extends TChannelName>(
    channel: T,
    listener: TClientEventListener<T>
  ): Promise<Result<TUnsubscribeFn>> {
    // `TClient` is strictly typed so we're fine casting the response as `any`.
    try {
      const unsubscribe = await listen<any>(channel, (event) => {
        listener(event.payload)
      })

      return Return.Value(unsubscribe)
    } catch (e) {
      return Return.Failed(e + "")
    }
  }

  public fetchPlatform(): Promise<TPlatform> {
    return os.platform()
  }

  public fetchArch(): Promise<TArch> {
    return os.arch()
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
}

// Singleton client
export const client = new Client()
