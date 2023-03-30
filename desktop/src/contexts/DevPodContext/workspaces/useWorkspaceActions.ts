import { useCallback, useSyncExternalStore } from "react"
import { TPublicAction } from "./action"
import { workspacesStore } from "./workspacesStore"

export function useWorkspaceActions(): readonly TPublicAction[] {
  const actions = useSyncExternalStore(
    useCallback((listener) => workspacesStore.subscribe(listener), []),
    () => workspacesStore.getAllActions()
  )

  return actions
}
