import { os } from "@tauri-apps/api"
import { listen } from "@tauri-apps/api/event"
import { TSettings } from "../contexts"
import { exists, noop, THandler } from "../lib"
import { Result, ResultError, Return } from "../lib/result"
import {
  TAddProviderConfig,
  TConfigureProviderConfig,
  TProviderID,
  TProviderOptions,
  TProviders,
  TUnsubscribeFn,
  TViewID,
  TWorkspace,
  TWorkspaceID,
  TWorkspaces,
  TWorkspaceStartConfig,
  TWorkspaceWithoutStatus,
} from "../types"
import { StartCommandCache } from "./cache"
import { TStreamEventListenerFn } from "./command"
import { ProviderCommands } from "./providerCommands"
import { WorkspaceCommands } from "./workspaceCommands"

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
  ): Promise<Result<TUnsubscribeFn>>
  fetchPlatform: () => Promise<TPlatform>
  fetchArch: () => Promise<TArch>
  workspaces: TWorkspaceClient
  providers: TProvidersClient
}>
type TClientSettings = Pick<TSettings, "debugFlag">

type TWorkspaceClient = Readonly<{
  listAll: () => Promise<Result<readonly TWorkspaceWithoutStatus[]>>
  getStatus: (workspaceID: TWorkspaceID) => Promise<Result<TWorkspace["status"]>>
  newID(rawWorkspaceSource: string): Promise<Result<TWorkspaceID>>
  start(
    workspaceID: TWorkspaceID,
    config: TWorkspaceStartConfig,
    viewID: TViewID,
    streamListener?: TStreamEventListenerFn
  ): Promise<Result<TWorkspace["status"]>>
  subscribeToStart: (
    workspaceID: TWorkspaceID,
    viewID: TViewID,
    streamListener?: TStreamEventListenerFn
  ) => TUnsubscribeFn
  stop(workspaceID: TWorkspaceID): Promise<Result<TWorkspace["status"]>>
  rebuild(workspaceID: TWorkspaceID): Promise<Result<TWorkspace["status"]>>
  remove(workspaceID: TWorkspaceID): Promise<ResultError>
}>

type TProvidersClient = Readonly<{
  listAll(): Promise<Result<TProviders>>
  newID(rawProviderSource: string): Promise<Result<TProviderID>>
  add(rawProviderSource: string, config: TAddProviderConfig): Promise<ResultError>
  remove(providerID: TProviderID): Promise<ResultError>
  getOptions(providerID: TProviderID): Promise<Result<TProviderOptions>>
  configure(providerID: TProviderID, config: TConfigureProviderConfig): Promise<ResultError>
}>

class Client implements TClient {
  private startCommandCache = new StartCommandCache()
  public workspaces = new WorkspacesClient(this.startCommandCache)
  public providers = new ProvidersClient()

  public setSetting<TSettingName extends keyof TClientSettings>(
    name: TSettingName,
    value: TSettings[TSettingName]
  ) {
    if (name === "debugFlag") {
      WorkspaceCommands.DEBUG = value
      ProviderCommands.DEBUG = value
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
}

class WorkspacesClient implements TWorkspaceClient {
  constructor(private startCommandCache: StartCommandCache) {}

  private createStartHandler(
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

  public async listAll(): Promise<Result<readonly TWorkspaceWithoutStatus[]>> {
    return WorkspaceCommands.ListWorkspaces()
  }

  public async getStatus(
    id: string
  ): Promise<Result<"Running" | "Busy" | "Stopped" | "NotFound" | null>> {
    const result = await WorkspaceCommands.GetWorkspaceStatus(id)
    if (result.err) {
      return result
    }

    const { status } = result.val

    return Return.Value(status)
  }

  public async newID(rawSource: string): Promise<Result<string>> {
    return await WorkspaceCommands.GetWorkspaceID(rawSource)
  }

  public async start(
    id: string,
    config: Readonly<{
      ideConfig?: { ide: string | null } | null | undefined
      providerConfig?: Readonly<{ providerID?: string | undefined }> | undefined
      sourceConfig?: Readonly<{ source: string }> | undefined
    }>,
    viewID: string,
    listener?: TStreamEventListenerFn | undefined
  ): Promise<Result<"Running" | "Busy" | "Stopped" | "NotFound" | null>> {
    const maybeRunningCommand = this.startCommandCache.get(id)
    const handler = this.createStartHandler(viewID, listener)

    // If `start` for id is running already,
    // wire up the new listener and return the existing operation
    if (exists(maybeRunningCommand)) {
      maybeRunningCommand.stream?.(handler)
      await maybeRunningCommand.promise

      return this.getStatus(id)
    }

    const cmd = WorkspaceCommands.StartWorkspace(id, config)
    const { operation, stream } = this.startCommandCache.connect(id, cmd)
    stream?.(handler)

    const result = await operation
    if (result.err) {
      return result
    }

    this.startCommandCache.clear(id)

    return this.getStatus(id)
  }

  public subscribeToStart(
    id: string,
    viewID: string,
    listener?: TStreamEventListenerFn | undefined
  ): TUnsubscribeFn {
    const maybeRunningCommand = this.startCommandCache.get(id)
    if (!exists(maybeRunningCommand)) {
      return noop
    }

    const maybeUnsubscribe = maybeRunningCommand.stream?.(this.createStartHandler(viewID, listener))

    return () => maybeUnsubscribe?.()
  }

  public async stop(
    id: string
  ): Promise<Result<"Running" | "Busy" | "Stopped" | "NotFound" | null>> {
    const result = await WorkspaceCommands.StopWorkspace(id).run()
    if (result.err) {
      return result
    }

    return this.getStatus(id)
  }

  public async rebuild(
    id: string
  ): Promise<Result<"Running" | "Busy" | "Stopped" | "NotFound" | null>> {
    const result = await WorkspaceCommands.RebuildWorkspace(id).run()
    if (result.err) {
      return result
    }

    return this.getStatus(id)
  }

  public async remove(id: string): Promise<ResultError> {
    return WorkspaceCommands.RemoveWorkspace(id).run()
  }
}

class ProvidersClient implements TProvidersClient {
  constructor() {}

  public async listAll(): Promise<Result<TProviders>> {
    return ProviderCommands.ListProviders()
  }

  public async newID(rawSource: string): Promise<Result<string>> {
    return ProviderCommands.GetProviderID(rawSource)
  }

  public async add(
    rawSource: string,
    config: Readonly<{ name?: string | null | undefined }>
  ): Promise<ResultError> {
    return ProviderCommands.AddProvider(rawSource, config)
  }

  public async remove(id: string): Promise<ResultError> {
    return ProviderCommands.RemoveProvider(id)
  }

  public async getOptions(id: string): Promise<Result<TProviderOptions>> {
    return ProviderCommands.GetProviderOptions(id)
  }

  public async configure(
    id: string,
    config: Readonly<{ options: Record<string, unknown>; useAsDefaultProvider: boolean }>
  ): Promise<ResultError> {
    const setResult = await ProviderCommands.SetProviderOptions(id, config.options)
    if (setResult.err) {
      return setResult
    }

    const initResult = await ProviderCommands.InitProvider(id)
    if (initResult.err) {
      return initResult
    }

    return Return.Ok()
  }
}

// Singleton client
export const client = new Client()
