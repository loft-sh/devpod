import { useEffect, useState } from "react"
import { TUnsubscribeFn } from "../types"
import { LocalStorageBackend, Store } from "./store"

const DEBUG_STORE_KEY = "debug"
const DEBUG_OPTIONS = ["commands", "actions", "workspaces"] as const
type TDebugOption = (typeof DEBUG_OPTIONS)[number]
type TDebug = Readonly<{
  isEnabled?: boolean
  toggle?(option: TDebugOption): Promise<void>
  get?(option: TDebugOption): Promise<boolean>
  print?(): void
}>
type TDebugStore = Record<TDebugOption, boolean>
type TInternalDebug = Readonly<{
  subscribe(option: TDebugOption, listener: (newValue: boolean) => void): TUnsubscribeFn
}>

function init(): TDebug & TInternalDebug {
  const store = new Store<TDebugStore>(new LocalStorageBackend<TDebugStore>(DEBUG_STORE_KEY))

  return {
    isEnabled: true,
    async toggle(option) {
      const current = (await store.get(option)) ?? false
      const newOptionValue = !current
      await store.set(option, newOptionValue)
    },
    async get(option) {
      return (await store.get(option)) ?? false
    },
    subscribe(option, listener) {
      return store.subscribe(option, listener)
    },
    print() {
      console.log(store)
    },
  }
}

const initialDebugOptions: TDebugStore = { commands: false, workspaces: false, actions: false }

type TUseDebug = Readonly<{ options: Record<TDebugOption, boolean> }> & Pick<TDebug, "isEnabled">
function useInternalDebug(): TUseDebug {
  const [options, setOptions] = useState<TDebugStore>(initialDebugOptions)

  useEffect(() => {
    ;(async () => {
      const initialOptions = await Promise.all(
        DEBUG_OPTIONS.map((option) =>
          Debug.get!(option)
            .then((value) => [option, value] as const)
            .catch(() => [option, false] as const)
        )
      )
      setOptions(
        initialOptions.reduce((acc, [key, value]) => ({ ...acc, [key]: value }), {} as TDebugStore)
      )
    })()
  }, [])

  useEffect(() => {
    const subscriptions: TUnsubscribeFn[] = []
    for (const option of DEBUG_OPTIONS) {
      subscriptions.push(
        (Debug as TInternalDebug).subscribe(option, (newValue) =>
          setOptions((currentOptions) => ({ ...currentOptions, [option]: newValue }))
        )
      )
    }

    return () => {
      for (const unsubscribe of subscriptions) {
        unsubscribe()
      }
    }
  }, [])

  return { options, isEnabled: true }
}
export function debug(option: TDebugOption, ...args: Parameters<(typeof console)["info"]>): void {
  Debug.get?.(option).then((isEnabled) => {
    if (isEnabled) {
      console.info(...args)
    }
  })
}

// Only available during development
export const Debug: TDebug = import.meta.env.DEV ? init() : { isEnabled: false }
// Only available during development
export const useDebug: typeof useInternalDebug = import.meta.env.DEV
  ? useInternalDebug
  : () => ({ options: initialDebugOptions, isEnabled: false })
