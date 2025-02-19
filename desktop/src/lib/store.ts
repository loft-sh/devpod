import { LazyStore } from "@tauri-apps/plugin-store"
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
      // noop
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

export class FileStorageBackend<T extends TBaseStore> implements TStorageBackend<T> {
  private readonly store: LazyStore

  constructor(name: string) {
    const fileName = `.${name}.json`
    this.store = new LazyStore(fileName)
  }

  public async set<TKey extends keyof T>(key: TKey, value: T[TKey]): Promise<void> {
    try {
      await this.store.set(key.toString(), value)
      await this.store.save()
    } catch {
      // noop
    }
  }

  public async get<TKey extends keyof T>(key: TKey): Promise<T[TKey] | null> {
    try {
      const maybeValue = await this.store.get<T[TKey] | null>(key.toString())
      if (!exists(maybeValue)) {
        return null
      }

      return maybeValue
    } catch {
      return null
    }
  }

  public async remove<TKey extends keyof T>(key: TKey): Promise<void> {
    await this.store.delete(key.toString())
    await this.store.save()
  }

  public async clear(): Promise<void> {
    await this.store.clear()
    await this.store.save()
  }
}

export class LocalStorageToFileMigrationBackend<T extends TBaseStore>
  implements TStorageBackend<T>
{
  private lsBackend: LocalStorageBackend<T>
  private fsBackend: FileStorageBackend<T>

  constructor(private storageKey: string) {
    this.lsBackend = new LocalStorageBackend<T>(this.storageKey)
    this.fsBackend = new FileStorageBackend<T>(this.storageKey)
  }

  public async set<TKey extends keyof T>(key: TKey, value: T[TKey]): Promise<void> {
    await this.fsBackend.set(key, value)
    // don't wait for removal to confirm
    try {
      this.lsBackend.remove(key)
    } catch {
      // noop
    }
  }

  public async get<TKey extends keyof T>(key: TKey): Promise<T[TKey] | null> {
    const fsValue = await this.fsBackend.get(key)
    if (exists(fsValue)) {
      return fsValue
    }

    const lsValue = await this.lsBackend.get(key)
    if (exists(lsValue)) {
      await this.fsBackend.set(key, lsValue)

      return lsValue
    }

    return null
  }

  public async remove<TKey extends keyof T>(key: TKey): Promise<void> {
    await Promise.all([this.lsBackend.remove(key), this.fsBackend.remove(key)])
  }

  public async clear(): Promise<void> {
    await Promise.all([this.lsBackend.clear(), this.fsBackend.clear()])
  }
}
