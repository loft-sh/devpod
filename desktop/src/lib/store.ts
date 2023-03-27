import { TUnsubscribeFn } from "../types"
import { EventManager } from "./eventManager"
import { exists } from "./helpers"

type TBaseStore = Record<string | number | symbol, unknown>

type TStore<T extends TBaseStore> = Readonly<{
  set<TKey extends keyof T>(key: TKey, value: T[TKey]): Promise<void>
  get<TKey extends keyof T>(key: TKey): Promise<T[TKey] | null>
  remove<TKey extends keyof T>(key: TKey): Promise<void>
  subscribe<TKey extends keyof T>(key: TKey, listener: (newValue: T[TKey]) => void): TUnsubscribeFn
  clear(): Promise<void>
}>
type TStorageBackend<T extends TBaseStore = TBaseStore> = Omit<TStore<T>, "subscribe">

export class Store<T extends TBaseStore> implements TStore<T> {
  private eventManager = new EventManager<T>()
  constructor(private backend: TStorageBackend<T>) {}

  public async set<TKey extends keyof T>(key: TKey, value: T[TKey]): Promise<void> {
    await this.backend.set(key, value)
    this.eventManager.publish(key, value)
  }

  public async get<TKey extends keyof T>(key: TKey): Promise<T[TKey] | null> {
    return this.backend.get(key)
  }

  public async remove<TKey extends keyof T>(key: TKey): Promise<void> {
    return this.backend.remove(key)
  }

  public subscribe<TKey extends keyof T>(
    key: TKey,
    listener: (newValue: T[TKey]) => void
  ): VoidFunction {
    const handler = EventManager.toHandler(listener)

    return this.eventManager.subscribe(key, handler)
  }

  public async clear(): Promise<void> {
    return this.backend.clear()
  }
}

export class LocalStorageBackend<T extends TBaseStore> implements TStorageBackend<T> {
  constructor(private storageKey: string) {}

  private getKey(key: keyof TBaseStore): string {
    return `devpod-${this.storageKey}-${key.toString()}`
  }

  public async set<TKey extends keyof T>(key: TKey, value: T[TKey]): Promise<void> {
    try {
      window.localStorage.setItem(this.getKey(key), JSON.stringify(value))
    } catch {
      // TODO: let caller know, noop for now
    }
  }

  public async get<TKey extends keyof T>(key: TKey): Promise<T[TKey] | null> {
    try {
      const maybeValue = window.localStorage.getItem(this.getKey(key))
      if (!exists(maybeValue)) {
        return null
      }

      return JSON.parse(maybeValue)
    } catch {
      return null
    }
  }

  public async remove<TKey extends keyof T>(key: TKey): Promise<void> {
    window.localStorage.removeItem(this.getKey(key))
  }

  public async clear(): Promise<void> {
    window.localStorage.clear()
  }
}
