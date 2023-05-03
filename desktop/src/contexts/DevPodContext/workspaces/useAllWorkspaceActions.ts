import { useCallback, useSyncExternalStore } from "react"
import { TActionObj } from "../action"
import { devPodStore } from "../devPodStore"

export function useAllWorkspaceActions() {
  const actions = useSyncExternalStore(
    useCallback((listener) => devPodStore.subscribe(listener), []),
    () => devPodStore.getAllActions()
  )

  return { active: actions.active, history: actions.history.slice().sort(sortByCreationDesc) }
}

function sortByCreationDesc(a: TActionObj, b: TActionObj) {
  return b.createdAt - a.createdAt
}
