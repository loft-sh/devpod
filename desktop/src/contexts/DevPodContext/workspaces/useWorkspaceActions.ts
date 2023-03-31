import { useCallback, useSyncExternalStore } from "react"
import { TActionObj } from "./action"
import { workspacesStore } from "./workspacesStore"

export function useWorkspaceActions() {
  const actions = useSyncExternalStore(
    useCallback((listener) => workspacesStore.subscribe(listener), []),
    () => workspacesStore.getAllActions()
  )

  return { active: actions.active, history: actions.history.slice().sort(sortByCreationDesc) }
}

function sortByCreationDesc(a: TActionObj, b: TActionObj) {
  return b.createdAt - a.createdAt
}
