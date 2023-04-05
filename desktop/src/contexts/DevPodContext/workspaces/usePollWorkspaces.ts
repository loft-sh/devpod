import { useEffect } from "react"
import { client } from "../../../client"
import { TWorkspaceID } from "../../../types"
import { REFETCH_INTERVAL_MS } from "../constants"
import { devpodStore } from "../devpodStore"

export function usePollWorkspaces() {
  useEffect(() => {
    const workspacesIntervalID = setInterval(async () => {
      const result = await client.workspaces.listAll()
      if (result.err) {
        return
      }
      devpodStore.setWorkspaces(result.val)
    }, REFETCH_INTERVAL_MS)

    const ongoingRequests: Record<TWorkspaceID, true> = {}
    const statusIntervalID = setInterval(async () => {
      for (const workspace of devpodStore.getAll()) {
        // Don't kick off a request if we already have one in flight or if we're executing an action on this workspace
        if (
          ongoingRequests[workspace.id] !== undefined ||
          devpodStore.getCurrentAction(workspace.id) !== undefined
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
          devpodStore.setStatus(workspace.id, result.val)
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
