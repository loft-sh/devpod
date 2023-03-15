import { invoke, os } from "@tauri-apps/api"
import { listen } from "@tauri-apps/api/event"
import {
  TProviders,
  TUnsubscribeFn,
  TWorkspace,
  TWorkspaceID,
  TWorkspaces,
  TWorkspaceStartConfig,
  TWorkspaceWithoutStatus,
} from "../types"
import {
  createCommand,
  listWorkspacesCommandConfig,
  rebuildWorkspaceCommandConfig,
  removeWorkspaceCommandConfig,
  startWorkspaceCommandConfig,
  stopWorkspaceCommandConfig,
  TStreamEventHandlerFn,
  workspaceStatusCommandConfig,
} from "./commands"
import { exists } from "../lib"

type TChannels = {
  providers: TProviders
  workspaces: TWorkspaces
}
type TChannelName = keyof TChannels
type TEventHandler<TChannel extends TChannelName> = (payload: TChannels[TChannel]) => void
export type TPlatform = Awaited<ReturnType<typeof os.platform>>
export type TArch = Awaited<ReturnType<typeof os.arch>>

type TClient = Readonly<{
  subscribe<T extends TChannelName>(
    channel: T,
    eventHandler: TEventHandler<T>
  ): Promise<TUnsubscribeFn>
  workspaces: TWorkspaceClient
  fetchPlatform: () => Promise<TPlatform>
  fetchArch: () => Promise<TArch>
}>

type TWorkspaceClient = Readonly<{
  listAll: () => Promise<readonly TWorkspaceWithoutStatus[]>
  getStatus: (workspaceID: TWorkspaceID) => Promise<TWorkspace["status"]>
  start(
    workspaceID: TWorkspaceID,
    config: TWorkspaceStartConfig,
    streamHandler?: TStreamEventHandlerFn
  ): Promise<TWorkspace["status"]>
  stop(workspaceID: TWorkspaceID): Promise<TWorkspace["status"]>
  rebuild(workspaceID: TWorkspaceID): Promise<TWorkspace["status"]>
  remove(workspaceID: TWorkspaceID): Promise<void>
  newWorkspaceID(rawWorkspaceSource: string): Promise<TWorkspaceID>
}>

function createClient(): TClient {
  return {
    async subscribe(channel, eventHandler) {
      // `TClient` is strictly typed so we're fine casting the response as `any`.
      const unsubscribe = await listen<any>(channel, (event) => {
        eventHandler(event.payload)
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
        return createCommand(listWorkspacesCommandConfig()).run()
      },
      async getStatus(id) {
        const { status } = await createCommand(workspaceStatusCommandConfig(id)).run()

        return status
      },
      async start(id, config, handler) {
        const cmd = createCommand(startWorkspaceCommandConfig(id, config))

        if (exists(handler)) {
          await cmd.stream(handler)
        } else {
          await cmd.run()
        }

        return this.getStatus(id)
      },
      async stop(id) {
        await createCommand(stopWorkspaceCommandConfig(id)).run()

        return this.getStatus(id)
      },
      async rebuild(id) {
        await createCommand(rebuildWorkspaceCommandConfig(id)).run()

        return this.getStatus(id)
      },
      async remove(id) {
        await createCommand(removeWorkspaceCommandConfig(id)).run()
      },
      async newWorkspaceID(rawSource) {
        return invoke<string>("new_workspace_id", { sourceName: rawSource })
      },
    },
  }
}

// singleton client
export const client = createClient()
