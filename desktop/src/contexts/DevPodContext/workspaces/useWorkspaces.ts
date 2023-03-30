import { useCallback, useSyncExternalStore } from "react"
import { TWorkspace } from "../../../types"
import { workspacesStore } from "./workspacesStore"

export function useWorkspaces(): readonly TWorkspace[] {
  const workspaces = useSyncExternalStore(
    useCallback((listener) => workspacesStore.subscribe(listener), []),
    () => workspacesStore.getAll()
  )

  return workspaces
}
