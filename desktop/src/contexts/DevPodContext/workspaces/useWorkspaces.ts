import { useCallback, useSyncExternalStore } from "react"
import { TWorkspace } from "../../../types"
import { devPodStore } from "../devPodStore"

export function useWorkspaces(): readonly TWorkspace[] {
  const workspaces = useSyncExternalStore(
    useCallback((listener) => devPodStore.subscribe(listener), []),
    () => devPodStore.getAll()
  )

  return workspaces
}
