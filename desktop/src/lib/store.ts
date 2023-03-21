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

export const Store = { create: createStore, backend: { createLocalStorageBackend } }

function createStore<T extends TBaseStore>(backend: TStorageBackend<T>): TStore<T> {
  const eventManager = EventManager.create<T>()

  return {
    async set(key, value) {
      await backend.set(key, value)
      eventManager.publish(key, value)
    },
    get(key) {
      return backend.get(key)
    },
    async remove(key) {
      return backend.remove(key)
    },
    async clear() {
      return backend.clear()
    },
    subscribe(key, listener) {
      const handler = EventManager.toHandler(listener)

      return eventManager.subscribe(key, handler)
    },
  }
}

function createLocalStorageBackend<T extends TBaseStore>(storageKey: string): TStorageBackend<T> {
  const getKey = (key: keyof TBaseStore) => `devpod-${storageKey}-${key.toString()}`

  return {
    async set(key, value) {
      try {
        window.localStorage.setItem(getKey(key), JSON.stringify(value))
      } catch {
        // TODO: let caller know, noop for now
      }
    },
    async get(key) {
      try {
        const maybeValue = window.localStorage.getItem(getKey(key))
        if (!exists(maybeValue)) {
          return null
        }

        return JSON.parse(maybeValue)
      } catch {
        return null
      }
    },
    async remove(key) {
      window.localStorage.removeItem(getKey(key))
    },
    async clear() {
      window.localStorage.clear()
    },
  }
}
