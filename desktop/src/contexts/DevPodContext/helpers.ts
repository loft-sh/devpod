import { UseMutationResult } from "@tanstack/react-query"
import {
  TOperationManager,
  TWithWorkspaceID,
  TWorkspace,
  TWorkspaceID,
  TWorkspaces,
} from "../../types"

export function getOperationManagerFromMutation<
  TRunConfig extends TWithWorkspaceID,
  TMutation extends UseMutationResult<unknown, unknown, TRunConfig, unknown>
>(mutation: TMutation): TOperationManager {
  return {
    run: (runConfig: TRunConfig) => {
      mutation.mutate(runConfig)
    },
    status: mutation.status,
    error: mutation.error,
  }
}

export function mergeWorkspaceStatus({
  id,
  status,
}: {
  id: TWorkspaceID
  status: TWorkspace["status"]
}) {
  return (workspaces: TWorkspaces | undefined): TWorkspaces | undefined => {
    return workspaces?.map((workspace) => {
      if (workspace.id !== id) {
        return workspace
      }

      return { ...workspace, status }
    })
  }
}

export function getWorkspacesStatusMap(
  workspaces: TWorkspaces | undefined
): Record<TWorkspaceID, TWorkspace["status"]> {
  return (workspaces ?? []).reduce((acc, curr) => ({ ...acc, [curr.id]: curr.status }), {})
}
