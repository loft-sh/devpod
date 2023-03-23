import { UseMutationResult } from "@tanstack/react-query"
import {
  TOperationManager,
  TWithResourceID,
  TWorkspace,
  TWorkspaceID,
  TWorkspaces,
} from "../../types"

export function getOperationManagerFromMutation<TRunConfig extends TWithResourceID>(
  mutation: UseMutationResult<unknown, unknown, TRunConfig, unknown>
): TOperationManager<TRunConfig> {
  return {
    run(runConfig) {
      mutation.mutate(runConfig)
    },
    status: mutation.status,
    error: mutation.error,
    target: mutation.variables,
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
