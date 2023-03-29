import { FileStorageBackend, Result, ResultError, Return, Store } from "../../lib"
import {
  TAddProviderConfig,
  TConfigureProviderConfig,
  TProviderID,
  TProviderOptions,
  TProviders,
} from "../../types"
import { TDebuggable } from "../types"
import { ProviderCommands } from "./providerCommands"

// WARN: These need to match the rust `file_name` and `dangling_provider_key` constants
// for reliable cleanup!
// Make sure to update them in `src/provider.rs` if you change them here!
const PROVIDERS_STORE_FILE_NAME = "providers"
const PROVIDERS_STORE_DANGLING_PROVIDER_KEY = "danglingProvider"

type TProviderStore = Readonly<{ [PROVIDERS_STORE_DANGLING_PROVIDER_KEY]: TProviderID | null }>

export class ProvidersClient implements TDebuggable {
  private readonly store = new Store<TProviderStore>(
    new FileStorageBackend<TProviderStore>(PROVIDERS_STORE_FILE_NAME)
  )
  private danglingProviderID: TProviderID | null = null
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

  public async add(rawSource: TProviderID, config: TAddProviderConfig): Promise<ResultError> {
    return ProviderCommands.AddProvider(rawSource, config)
  }

  public async remove(id: TProviderID): Promise<ResultError> {
    return ProviderCommands.RemoveProvider(id)
  }

  public async getOptions(id: TProviderID): Promise<Result<TProviderOptions>> {
    return ProviderCommands.GetProviderOptions(id)
  }

  public async configure(
    id: TProviderID,
    { useAsDefaultProvider, initializeProvider, reuseMachine, options }: TConfigureProviderConfig
  ): Promise<ResultError> {
    if (useAsDefaultProvider) {
      return ProviderCommands.UseProvider(id, options, reuseMachine)
    }

    const setResult = await ProviderCommands.SetProviderOptions(id, options, reuseMachine)
    if (setResult.err) {
      return setResult
    }

    if (initializeProvider) {
      const initResult = await ProviderCommands.InitProvider(id)
      if (initResult.err) {
        return initResult
      }
    }

    return Return.Ok()
  }

  public async initialize(id: TProviderID): Promise<ResultError> {
    return ProviderCommands.InitProvider(id)
  }

  public setDangling(id: TProviderID) {
    this.danglingProviderID = id
    this.storeOperationQueue = this.storeOperationQueue.then(() =>
      this.store.set("danglingProvider", id)
    )
  }

  public popDangling(): TProviderID | null {
    const maybeProviderID = this.danglingProviderID
    this.danglingProviderID = null
    this.storeOperationQueue = this.storeOperationQueue.then(() =>
      this.store.remove("danglingProvider")
    )

    return maybeProviderID
  }
}
