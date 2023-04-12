import { useEffect } from "react"
import { client } from "../../../client"
import { TWorkspaceID } from "../../../types"
import { REFETCH_INTERVAL_MS } from "../constants"
import { devPodStore } from "../devPodStore"

export function usePollWorkspaces() {
  useEffect(() => {
    const workspacesIntervalID = setInterval(async () => {
      const result = await client.workspaces.listAll()
      if (result.err) {
        return
      }
      devPodStore.setWorkspaces(result.val)
    }, REFETCH_INTERVAL_MS)

    const ongoingRequests: Record<TWorkspaceID, true> = {}
    const statusIntervalID = setInterval(async () => {
      for (const workspace of devPodStore.getAll()) {
        // Don't kick off a request if we already have one in flight or if we're executing an action on this workspace
        if (
          ongoingRequests[workspace.id] !== undefined ||
          devPodStore.getCurrentAction(workspace.id) !== undefined
        ) {
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
          devPodStore.setStatus(workspace.id, result.val)
        } finally {
          delete ongoingRequests[workspace.id]
        }
      }
    }, REFETCH_INTERVAL_MS)

    return () => {
      clearInterval(workspacesIntervalID)
      clearInterval(statusIntervalID)
    }
  }, [])
}
