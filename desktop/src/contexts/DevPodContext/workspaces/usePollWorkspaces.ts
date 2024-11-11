import { useCallback, useEffect } from "react"
import { client } from "../../../client"
import { TWorkspaceID } from "../../../types"
import { REFETCH_INTERVAL_MS } from "../constants"
import { WorkspaceStore, useWorkspaceStore } from "../workspaceStore"

export function usePollWorkspaces() {
  const { store } = useWorkspaceStore<WorkspaceStore>()
  const listWorkspaces = useCallback(async () => {
    const result = await client.workspaces.listAll()
    if (result.err) {
      return
    }
    store.setWorkspaces(result.val)
  }, [store])

  const updateStatus = useCallback(
    async (ongoingRequests: Record<TWorkspaceID, true>) => {
      for (const workspace of store.getAll()) {
        // Don't kick off a request if we already have one in flight or if we're executing an action on this workspace
        const currentAction = store.getCurrentAction(workspace.id)
        if (ongoingRequests[workspace.id] !== undefined || currentAction != undefined) {
          continue
        }

        ongoingRequests[workspace.id] = true
        try {
          const result = await client.workspaces.getStatus(workspace.id)
          if (result.err) {
            continue
          }
          // We don't care about the order here, we just want to update the status
          // whenever we get a result back
          store.setStatus(workspace.id, result.val)
        } finally {
          delete ongoingRequests[workspace.id]
        }
      }
    },
    [store]
  )

  useEffect(() => {
    const workspacesIntervalID = setInterval(listWorkspaces, REFETCH_INTERVAL_MS)

    const ongoingRequests: Record<TWorkspaceID, true> = {}
    const statusIntervalID = setInterval(async () => {
      await updateStatus(ongoingRequests)
    }, REFETCH_INTERVAL_MS)

    const initialTimeoutID = setTimeout(async () => {
      await listWorkspaces()
      await updateStatus(ongoingRequests)
    }, 0)

    return () => {
      clearInterval(workspacesIntervalID)
      clearInterval(statusIntervalID)
      clearTimeout(initialTimeoutID)
    }
  }, [listWorkspaces, updateStatus])
}
