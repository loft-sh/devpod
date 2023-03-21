import { invoke, os } from "@tauri-apps/api"
import { listen } from "@tauri-apps/api/event"
import { TSettings } from "../contexts"
import { exists, noop, THandler } from "../lib"
import {
  TProviders,
  TUnsubscribeFn,
  TViewID,
  TWorkspace,
  TWorkspaceID,
  TWorkspaces,
  TWorkspaceStartConfig,
  TWorkspaceWithoutStatus,
} from "../types"
import { createStartCommandCache } from "./cache"
import {
  createCommand,
  createWithDebug,
  listWorkspacesCommandConfig,
  rebuildWorkspaceCommandConfig,
  removeWorkspaceCommandConfig,
  startWorkspaceCommandConfig,
  stopWorkspaceCommandConfig,
  TStreamEventListenerFn,
  workspaceStatusCommandConfig,
} from "./commands"
import { DEFAULT_STATIC_COMMAND_CONFIG } from "./constants"

type TChannels = {
  providers: TProviders
  workspaces: TWorkspaces
}
type TChannelName = keyof TChannels
type TClientEventListener<TChannel extends TChannelName> = (payload: TChannels[TChannel]) => void
export type TPlatform = Awaited<ReturnType<typeof os.platform>>
export type TArch = Awaited<ReturnType<typeof os.arch>>

type TClient = Readonly<{
  setSetting<TSettingName extends keyof TClientSettings>(
    name: TSettingName,
    settingValue: TSettings[TSettingName]
  ): void
  subscribe<T extends TChannelName>(
    channel: T,
    eventListener: TClientEventListener<T>
  ): Promise<TUnsubscribeFn>
  workspaces: TWorkspaceClient
  fetchPlatform: () => Promise<TPlatform>
  fetchArch: () => Promise<TArch>
}>
type TClientSettings = Pick<TSettings, "debugFlag">

type TWorkspaceClient = Readonly<{
  listAll: () => Promise<readonly TWorkspaceWithoutStatus[]>
  getStatus: (workspaceID: TWorkspaceID) => Promise<TWorkspace["status"]>
  start(
    workspaceID: TWorkspaceID,
    config: TWorkspaceStartConfig,
    viewID: TViewID,
    streamListener?: TStreamEventListenerFn
  ): Promise<TWorkspace["status"]>
  subscribeToStart: (
    workspaceID: TWorkspaceID,
    viewID: TViewID,
    streamListener?: TStreamEventListenerFn
  ) => TUnsubscribeFn
  stop(workspaceID: TWorkspaceID): Promise<TWorkspace["status"]>
  rebuild(workspaceID: TWorkspaceID): Promise<TWorkspace["status"]>
  remove(workspaceID: TWorkspaceID): Promise<void>
  newWorkspaceID(rawWorkspaceSource: string): Promise<TWorkspaceID>
}>

function createClient(): TClient {
  const startCommandCache = createStartCommandCache()
  const settings = new Map<keyof TClientSettings, TClientSettings[keyof TClientSettings]>([
    ["debugFlag", DEFAULT_STATIC_COMMAND_CONFIG.debug],
  ])
  const withDebug = createWithDebug(
    () => settings.get("debugFlag") ?? DEFAULT_STATIC_COMMAND_CONFIG.debug
  )

  return {
    setSetting(name, value) {
      settings.set(name, value)
    },
    async subscribe(channel, listener) {
      // `TClient` is strictly typed so we're fine casting the response as `any`.
      const unsubscribe = await listen<any>(channel, (event) => {
        listener(event.payload)
      })

      return unsubscribe
    },
    fetchPlatform() {
      return os.platform()
    },
    fetchArch() {
      return os.arch()
    },
    workspaces: {
      async listAll() {
        return createCommand(withDebug(listWorkspacesCommandConfig())).then((command) =>
          command.run()
        )
      },
      async getStatus(id) {
        // Don't run with `--debug` flag!
        const { status } = await createCommand(workspaceStatusCommandConfig(id)).then((command) =>
          command.run()
        )

        return status
      },
      subscribeToStart(id, viewID, listener) {
        const maybeRunningCommand = startCommandCache.get(id)
        if (!exists(maybeRunningCommand)) {
          return noop
        }

        const maybeUnsubscribe = maybeRunningCommand.stream?.(createStartHandler(viewID, listener))

        return () => maybeUnsubscribe?.()
      },
      async start(id, config, viewID, listener) {
        try {
          const maybeRunningCommand = startCommandCache.get(id)
          const handler = createStartHandler(viewID, listener)

          // If `start` for id is running already,
          // wire up the new listener and return the existing operation
          if (exists(maybeRunningCommand)) {
            maybeRunningCommand.stream?.(handler)
            await maybeRunningCommand.promise

            return this.getStatus(id)
          }

          const cmd = await createCommand(withDebug(startWorkspaceCommandConfig(id, config)))
          const { operation, stream } = startCommandCache.connect(id, cmd)
          stream?.(handler)

          await operation
        } finally {
          startCommandCache.clear(id)
        }

        return this.getStatus(id)
      },
      async stop(id) {
        await createCommand(withDebug(stopWorkspaceCommandConfig(id))).then((command) =>
          command.run()
        )

        return this.getStatus(id)
      },
      async rebuild(id) {
        await createCommand(withDebug(rebuildWorkspaceCommandConfig(id))).then((command) =>
          command.run()
        )

        return this.getStatus(id)
      },
      async remove(id) {
        await createCommand(withDebug(removeWorkspaceCommandConfig(id))).then((command) =>
          command.run()
        )
      },
      async newWorkspaceID(rawSource) {
        return invoke<string>("new_workspace_id", { sourceName: rawSource })
      },
    },
  }
}

// singleton client
export const client = createClient()

function createStartHandler(
  viewID: TViewID,
  listener: TStreamEventListenerFn | undefined
): THandler<TStreamEventListenerFn> {
  return {
    id: viewID,
    eq(other) {
      return viewID === other.id
    },
    notify: exists(listener) ? listener : noop,
  }
}
