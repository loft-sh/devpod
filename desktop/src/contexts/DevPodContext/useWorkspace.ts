import { useQuery, useQueryClient } from "@tanstack/react-query"
import { exists } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import { TQueryResult, TWorkspace, TWorkspaceID, TWorkspaceManager, TWorkspaces } from "../../types"
import { useWorkspaceManager } from "./useWorkspaceManager"

export function useWorkspace(
  workspaceID: TWorkspaceID | undefined
): [TQueryResult<TWorkspace>, TWorkspaceManager] {
  const queryClient = useQueryClient()
  const manager = useWorkspaceManager()

  const { data, status, error } = useQuery({
    queryKey: QueryKeys.workspace(workspaceID!),
    queryFn: ({ queryKey }) => {
      const [maybeWorkspacesKey] = queryKey

      if (!exists(maybeWorkspacesKey)) {
        throw Error(`Workspace ${workspaceID} not found`)
      }

      const maybeWorkspace = queryClient
        .getQueryData<TWorkspaces>([maybeWorkspacesKey])
        ?.find(({ id }) => workspaceID === id)

      if (!exists(maybeWorkspace)) {
        throw Error(`Workspace ${workspaceID} not found`)
      }

      return maybeWorkspace
    },
    enabled: exists(workspaceID),
  })

  return [[data, { status, error }], manager]
}
