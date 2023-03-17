import { useQueryClient } from "@tanstack/react-query"
import { useCallback } from "react"
import { exists } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import { TWorkspace, TWorkspaces } from "../../types"
import { TDevpodContext } from "./DevPodProvider"
import { mergeWorkspaceStatus } from "./helpers"

export function useUpdateWorkspaceStatus() {
  const queryClient = useQueryClient()
  const updateWorkspaceStatus = useCallback<TDevpodContext["updateWorkspaceStatus"]>(
    (newStatus, { workspaceID }) => {
      // update cache with new status
      queryClient.setQueryData<TWorkspaces>(
        QueryKeys.WORKSPACES,
        mergeWorkspaceStatus({ id: workspaceID, status: newStatus })
      )

      const workspaceKey = QueryKeys.workspace(workspaceID)
      queryClient.setQueryData<TWorkspace>(workspaceKey, (currentWorkspace) =>
        exists(currentWorkspace) ? { ...currentWorkspace, status: newStatus } : currentWorkspace
      )
      const workspaceStatusKey = QueryKeys.workspaceStatus(workspaceID)
      queryClient.setQueryData(workspaceStatusKey, newStatus)
    },
    [queryClient]
  )

  return updateWorkspaceStatus
}
