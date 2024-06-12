import { FileStorageBackend, Result, ResultError, Return, Store, isEmpty } from "../../lib"
import {
  TAddProviderConfig,
  TCheckProviderUpdateResult,
  TConfigureProviderConfig,
  TProviderID,
  TProviderOptions,
  TProviderSource,
  TProviders,
} from "../../types"
import { TDebuggable } from "../types"
import { ProviderCommands } from "./providerCommands"

// WARN: These need to match the rust `file_name` and `dangling_provider_key` constants
// for reliable cleanup!
// Make sure to update them in `src/provider.rs` if you change them here!
const PROVIDERS_STORE_FILE_NAME = "providers"
const PROVIDERS_STORE_DANGLING_PROVIDER_KEY = "danglingProviders"

type TProviderStore = Readonly<{ [PROVIDERS_STORE_DANGLING_PROVIDER_KEY]: readonly TProviderID[] }>

export class ProvidersClient implements TDebuggable {
  private readonly store = new Store<TProviderStore>(
    new FileStorageBackend<TProviderStore>(PROVIDERS_STORE_FILE_NAME)
  )
  private danglingProviderIDs: TProviderID[] = []
  // Queues store operations and guarantees they will be executed in order
  private storeOperationQueue: Promise<unknown> = Promise.resolve()

  constructor() {}

  public setDebug(isEnabled: boolean): void {
    ProviderCommands.DEBUG = isEnabled
  }

  public async listAll(): Promise<Result<TProviders>> {
    return ProviderCommands.ListProviders()
  }

  public async newID(rawSource: string): Promise<Result<string>> {
    return ProviderCommands.GetProviderID(rawSource)
  }

  public async checkUpdate(id: TProviderID): Promise<Result<TCheckProviderUpdateResult>> {
    return ProviderCommands.CheckProviderUpdate(id)
  }

  public async update(id: TProviderID, source: TProviderSource): Promise<Result<void>> {
    return ProviderCommands.UpdateProvider(id, source)
  }

  public async add(rawSource: TProviderID, config: TAddProviderConfig): Promise<ResultError> {
    return ProviderCommands.AddProvider(rawSource, config)
  }

  public async remove(id: TProviderID): Promise<ResultError> {
    return ProviderCommands.RemoveProvider(id)
  }

  public async getOptions(id: TProviderID): Promise<Result<TProviderOptions>> {
    return ProviderCommands.GetProviderOptions(id)
  }

  public async useProvider(id: TProviderID): Promise<ResultError> {
    return ProviderCommands.UseProvider(id)
  }

  public async setOptionsDry(
    id: TProviderID,
    { options, reconfigure }: TConfigureProviderConfig
  ): Promise<Result<TProviderOptions | undefined>> {
    return ProviderCommands.SetProviderOptions(id, options, false, true, reconfigure)
  }

  public async configure(
    id: TProviderID,
    { useAsDefaultProvider, reuseMachine, options }: TConfigureProviderConfig
  ): Promise<ResultError> {
    const setResult = await ProviderCommands.SetProviderOptions(id, options, !!reuseMachine)
    if (setResult.err) {
      return setResult as ResultError
    }

    if (useAsDefaultProvider) {
      return ProviderCommands.UseProvider(id)
    }

    return Return.Ok()
  }

  public setDangling(id: TProviderID): void {
    this.danglingProviderIDs.push(id)
    const ids = this.danglingProviderIDs.slice()
    this.storeOperationQueue = this.storeOperationQueue.then(() =>
      this.store.set("danglingProviders", ids)
    )
  }

  public popAllDangling(): readonly TProviderID[] {
    const maybeProviderIDs = this.danglingProviderIDs.slice()
    this.danglingProviderIDs.length = 0
    this.storeOperationQueue = this.storeOperationQueue.then(() =>
      this.store.remove("danglingProviders")
    )

    return maybeProviderIDs
  }

  public popDangling(): TProviderID | undefined {
    const lastProviderID = this.danglingProviderIDs.pop()
    const ids = this.danglingProviderIDs.slice()
    this.storeOperationQueue = this.storeOperationQueue.then(() => {
      if (isEmpty(ids)) {
        return this.store.remove("danglingProviders")
      }

      return this.store.set("danglingProviders", ids)
    })

    return lastProviderID
  }
}
