import { useCallback, useSyncExternalStore } from "react"
import { IWorkspaceStore, useWorkspaceStore } from "../workspaceStore"

export function useWorkspaces<TW>(): readonly TW[] {
  const { store } = useWorkspaceStore<IWorkspaceStore<string, TW>>()
  const workspaces = useSyncExternalStore(
    useCallback((listener) => store.subscribe(listener), [store]),
    () => store.getAll()
  )

  return workspaces
}
