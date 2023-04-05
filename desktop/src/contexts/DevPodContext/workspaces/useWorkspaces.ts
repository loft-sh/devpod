import { useCallback, useSyncExternalStore } from "react"
import { TWorkspace } from "../../../types"
import { devpodStore } from "../devpodStore"

export function useWorkspaces(): readonly TWorkspace[] {
  const workspaces = useSyncExternalStore(
    useCallback((listener) => devpodStore.subscribe(listener), []),
    () => devpodStore.getAll()
  )

  return workspaces
}
