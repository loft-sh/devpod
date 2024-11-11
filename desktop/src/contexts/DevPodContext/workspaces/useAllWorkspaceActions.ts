import { useCallback, useSyncExternalStore } from "react"
import { TActionObj } from "../action"
import { useWorkspaceStore } from "../workspaceStore"

export function useAllWorkspaceActions() {
  const { store } = useWorkspaceStore()
  const actions = useSyncExternalStore(
    useCallback((listener) => store.subscribe(listener), [store]),
    () => store.getAllActions()
  )

  return { active: actions.active, history: actions.history.slice().sort(sortByCreationDesc) }
}

function sortByCreationDesc(a: TActionObj, b: TActionObj) {
  return b.createdAt - a.createdAt
}
