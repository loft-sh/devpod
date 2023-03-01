import { listen } from "@tauri-apps/api/event"
import { createContext, ReactNode, useContext, useEffect, useMemo, useState } from "react"

type TOption = Readonly<{ local: string | null; retrieved: string | null; value: string | null }>
export type TProviders = Readonly<{
  defaultProvider: string
  providers: Record<string, { options: Record<string, TOption> | null }>
}>
export type TWorkspace = Readonly<{
  id: string | null
  provider: Readonly<{ name: string | null }> | null
}>
export type TWorkspaces = readonly TWorkspace[]
type TDevpodContext = Readonly<{
  providers: TProviders | null
  workspaces: TWorkspaces | null
}>
const DevpodContext = createContext<TDevpodContext>(null!)

export function DevpodProvider({ children }: Readonly<{ children?: ReactNode }>) {
  const [providers, setProviders] = useState<TProviders | null>(null)
  const [workspaces, setWorkspaces] = useState<TWorkspaces | null>(null)

  useEffect(() => {
    ;(async () => {
      const unsubscribe = await listen<TProviders>("providers", (event) => {
        setProviders(event.payload)
      })

      return unsubscribe
    })()
  }, [])

  useEffect(() => {
    ;(async () => {
      const unsubscribe = await listen<TWorkspaces>("workspaces", (event) => {
        setWorkspaces(event.payload)
      })

      return unsubscribe
    })()
  }, [])

  const value = useMemo<TDevpodContext>(() => ({ providers, workspaces }), [providers, workspaces])

  return <DevpodContext.Provider value={value}>{children}</DevpodContext.Provider>
}

export function useWorkspaces(): TDevpodContext["workspaces"] {
  return useContext(DevpodContext).workspaces
}

export function useProviders(): TDevpodContext["providers"] {
  return useContext(DevpodContext).providers
}
