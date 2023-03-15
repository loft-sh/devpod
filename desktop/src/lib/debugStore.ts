import { useEffect, useState } from "react"
import { TUnsubscribeFn } from "../types"
import { deserializeMap, exists, serializeMap } from "./helpers"

const DEBUG_OPTIONS = ["logs"] as const
type TDebugOption = (typeof DEBUG_OPTIONS)[number]
type TDebug = Readonly<{
  isEnabled?: boolean
  toggle?(option: TDebugOption): void
  get?(option: TDebugOption): boolean
  print?(): void
}>
type TDebugStore = Map<TDebugOption, boolean>
type THandler = (newValue: boolean) => void
type TInternalDebug = Readonly<{
  // WARN: Only supports one subscriber per option.
  subscribe(option: TDebugOption, handler: THandler): TUnsubscribeFn
}>
const DEBUG_STORE_KEY = "devpod-debug-store"

function initStore(): TDebugStore {
  const maybeSerializedStore = localStorage.getItem(DEBUG_STORE_KEY)
  if (!exists(maybeSerializedStore)) {
    return new Map<TDebugOption, boolean>()
  }

  return deserializeMap<TDebugStore>(maybeSerializedStore)
}

function init(): TDebug & TInternalDebug {
  const listeners = new Map<TDebugOption, THandler>()
  const store = initStore()

  return {
    isEnabled: true,
    toggle(option) {
      const current = store.get(option) ?? false
      const newOptionValue = !current
      store.set(option, newOptionValue)

      // notify subscribers
      listeners.get(option)?.(newOptionValue)
      // persist store to local storage
      localStorage.setItem(DEBUG_STORE_KEY, serializeMap(store))
    },
    get(option) {
      return store.get(option) ?? false
    },
    subscribe(option, handler) {
      listeners.set(option, handler)

      return () => listeners.delete(option)
    },
    print() {
      console.log(store)
    },
  }
}

function getInitialDebugOptions() {
  return DEBUG_OPTIONS.reduce(
    (acc, curr) => ({ ...acc, [curr]: Debug.get!(curr) }), // we know `Debug` will be enabled when this hook is
    {} as Record<TDebugOption, boolean>
  )
}

function useInternalDebug(): Record<TDebugOption, boolean> {
  const [options, setOptions] = useState<Record<TDebugOption, boolean>>(getInitialDebugOptions)

  useEffect(() => {
    const subscriptions: VoidFunction[] = []
    const handler = (option: TDebugOption) => (newValue: boolean) => {
      setOptions((currentOptions) => ({ ...currentOptions, [option]: newValue }))
    }

    for (const option of DEBUG_OPTIONS) {
      subscriptions.push((Debug as TInternalDebug).subscribe(option, handler(option)))
    }

    return () => {
      for (const unsubscribe of subscriptions) {
        unsubscribe()
      }
    }
  }, [])

  return options
}

// Only available during development
export const Debug: TDebug = import.meta.env.DEV ? init() : { isEnabled: false }
// Only available during development
export const useDebug: typeof useInternalDebug = import.meta.env.DEV
  ? useInternalDebug
  : () => getInitialDebugOptions()
