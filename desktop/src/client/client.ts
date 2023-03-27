import { os } from "@tauri-apps/api"
import { listen } from "@tauri-apps/api/event"
import { TSettings } from "../contexts"
import { exists, noop, THandler } from "../lib"
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
import {
  addProviderCommandConfig,
  createCommand,
  createWithDebug,
  getProviderOptionsCommandConfig,
  initProviderCommandConfig,
  listProvidersCommandConfig,
  listWorkspacesCommandConfig,
  providerIDCommandConfig,
  rebuildWorkspaceCommandConfig,
  removeProviderCommandConfig,
  removeWorkspaceCommandConfig,
  setProviderOptionsCommandConfig,
  startWorkspaceCommandConfig,
  stopWorkspaceCommandConfig,
  TStreamEventListenerFn,
  useProviderCommandConfig,
  workspaceIDCommandConfig,
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
  fetchPlatform: () => Promise<TPlatform>
  fetchArch: () => Promise<TArch>
  workspaces: TWorkspaceClient
  providers: TProvidersClient
}>
type TClientSettings = Pick<TSettings, "debugFlag">

type TWorkspaceClient = Readonly<{
  listAll: () => Promise<readonly TWorkspaceWithoutStatus[]>
  getStatus: (workspaceID: TWorkspaceID) => Promise<TWorkspace["status"]>
  newID(rawWorkspaceSource: string): Promise<TWorkspaceID>
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
}>

type TProvidersClient = Readonly<{
  listAll(): Promise<TProviders>
  newID(rawProviderSource: string): Promise<TProviderID>
  add(rawProviderSource: string, config: TAddProviderConfig): Promise<void>
  remove(providerID: TProviderID): Promise<void>
  getOptions(providerID: TProviderID): Promise<TProviderOptions>
  configure(providerID: TProviderID, config: TConfigureProviderConfig): Promise<void>
}>

class Client implements TClient {
  private startCommandCache = new StartCommandCache()
  private settings = new Map<keyof TClientSettings, TClientSettings[keyof TClientSettings]>([
    ["debugFlag", DEFAULT_STATIC_COMMAND_CONFIG.debug],
  ])

  private withDebug = createWithDebug(
    () => this.settings.get("debugFlag") ?? DEFAULT_STATIC_COMMAND_CONFIG.debug
  )
  public workspaces = new WorkspacesClient(this.withDebug, this.startCommandCache)
  public providers = new ProvidersClient(this.withDebug)

  public setSetting<TSettingName extends keyof TClientSettings>(
    name: TSettingName,
    value: TSettings[TSettingName]
  ) {
    this.settings.set(name, value)
  }
  public async subscribe<T extends TChannelName>(
    channel: T,
    listener: TClientEventListener<T>
  ): Promise<TUnsubscribeFn> {
    // `TClient` is strictly typed so we're fine casting the response as `any`.
    const unsubscribe = await listen<any>(channel, (event) => {
      listener(event.payload)
    })

    return unsubscribe
  }

  public fetchPlatform(): Promise<TPlatform> {
    return os.platform()
  }

  public fetchArch(): Promise<TArch> {
    return os.arch()
  }
}

class WorkspacesClient implements TWorkspaceClient {
  constructor(
    private withDebug: ReturnType<typeof createWithDebug>,
    private startCommandCache: StartCommandCache
  ) {}

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

  public async listAll(): Promise<readonly TWorkspaceWithoutStatus[]> {
    return createCommand(listWorkspacesCommandConfig()).then((cmd) => cmd.run())
  }

  public async getStatus(id: string): Promise<"Running" | "Busy" | "Stopped" | "NotFound" | null> {
    // Don't run with `--debug` flag!
    const { status } = await createCommand(workspaceStatusCommandConfig(id)).then((cmd) =>
      cmd.run()
    )

    return status
  }

  public async newID(rawSource: string): Promise<string> {
    return createCommand(workspaceIDCommandConfig(rawSource)).then((cmd) => cmd.run())
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
  ): Promise<"Running" | "Busy" | "Stopped" | "NotFound" | null> {
    try {
      const maybeRunningCommand = this.startCommandCache.get(id)
      const handler = this.createStartHandler(viewID, listener)

      // If `start` for id is running already,
      // wire up the new listener and return the existing operation
      if (exists(maybeRunningCommand)) {
        maybeRunningCommand.stream?.(handler)
        await maybeRunningCommand.promise

        return this.getStatus(id)
      }

      const cmd = await createCommand(this.withDebug(startWorkspaceCommandConfig(id, config)))
      const { operation, stream } = this.startCommandCache.connect(id, cmd)
      stream?.(handler)

      await operation
    } finally {
      this.startCommandCache.clear(id)
    }

    return this.getStatus(id)
  }
  public subscribeToStart(
    id: string,
    viewID: string,
    listener?: TStreamEventListenerFn | undefined
  ): VoidFunction {
    const maybeRunningCommand = this.startCommandCache.get(id)
    if (!exists(maybeRunningCommand)) {
      return noop
    }

    const maybeUnsubscribe = maybeRunningCommand.stream?.(this.createStartHandler(viewID, listener))

    return () => maybeUnsubscribe?.()
  }

  public async stop(id: string): Promise<"Running" | "Busy" | "Stopped" | "NotFound" | null> {
    await createCommand(this.withDebug(stopWorkspaceCommandConfig(id))).then((cmd) => cmd.run())

    return this.getStatus(id)
  }

  public async rebuild(id: string): Promise<"Running" | "Busy" | "Stopped" | "NotFound" | null> {
    await createCommand(this.withDebug(rebuildWorkspaceCommandConfig(id))).then((cmd) => cmd.run())

    return this.getStatus(id)
  }
  public async remove(id: string): Promise<void> {
    await createCommand(this.withDebug(removeWorkspaceCommandConfig(id))).then((cmd) => cmd.run())
  }
}
class ProvidersClient implements TProvidersClient {
  constructor(private withDebug: ReturnType<typeof createWithDebug>) {}

  public async listAll(): Promise<TProviders> {
    return createCommand(listProvidersCommandConfig()).then((cmd) => cmd.run())
  }

  public async newID(rawSource: string): Promise<string> {
    return createCommand(providerIDCommandConfig(rawSource)).then((cmd) => cmd.run())
  }

  public async add(
    rawSource: string,
    config: Readonly<{ name?: string | null | undefined }>
  ): Promise<void> {
    return createCommand(addProviderCommandConfig(rawSource, config)).then((cmd) => cmd.run())
  }

  public async remove(id: string): Promise<void> {
    await createCommand(this.withDebug(removeProviderCommandConfig(id))).then((cmd) => cmd.run())
  }

  public async getOptions(id: string): Promise<TProviderOptions> {
    return createCommand(getProviderOptionsCommandConfig(id)).then((cmd) => cmd.run())
  }

  public async configure(
    id: string,
    config: Readonly<{ options: Record<string, unknown>; useAsDefaultProvider: boolean }>
  ): Promise<void> {
    if (config.useAsDefaultProvider) {
      // eslint-disable-next-line react-hooks/rules-of-hooks
      return createCommand(useProviderCommandConfig(id, config.options)).then((cmd) => cmd.run())
    } else {
      const setProviderOptionsCmd = await createCommand(
        setProviderOptionsCommandConfig(id, config.options)
      )
      const initProviderCmd = await createCommand(initProviderCommandConfig(id))

      await setProviderOptionsCmd.run()
      await initProviderCmd.run()

      return
    }
  }
}

// Singleton client
export const client = new Client()
