import { useCallback, useSyncExternalStore } from "react"
import { TActionObj } from "../action"
import { devpodStore } from "../devpodStore"

export function useWorkspaceActions() {
  const actions = useSyncExternalStore(
    useCallback((listener) => devpodStore.subscribe(listener), []),
    () => devpodStore.getAllActions()
  )

  return { active: actions.active, history: actions.history.slice().sort(sortByCreationDesc) }
}

function sortByCreationDesc(a: TActionObj, b: TActionObj) {
  return b.createdAt - a.createdAt
}
