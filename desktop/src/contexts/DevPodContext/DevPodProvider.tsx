import { useIsMutating, useQueries, useQuery, useQueryClient } from "@tanstack/react-query"
import { createContext, ReactNode, useMemo } from "react"
import { client } from "../../client"
import { exists } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import { TProviders, TQueryResult, TWithWorkspaceID, TWorkspace, TWorkspaces } from "../../types"
import { useUpdateWorkspaceStatus } from "./useUpdateWorkspaceStatus"

export type TDevpodContext = Readonly<{
  providers: TQueryResult<TProviders>
  workspaces: TQueryResult<TWorkspaces>
  updateWorkspaceStatus: (newStatus: TWorkspace["status"], config: TWithWorkspaceID) => void
}>
export const DevPodContext = createContext<TDevpodContext>(null!)

const REFETCH_INTERVAL_MS = 1_000

export function DevPodProvider({ children }: Readonly<{ children?: ReactNode }>) {
  const queryClient = useQueryClient()
  const providersQuery = useQuery({
    queryKey: QueryKeys.PROVIDERS,
    queryFn: () => client.providers.listAll(),
    refetchInterval: REFETCH_INTERVAL_MS,
  })

  const updateWorkspaceStatus = useUpdateWorkspaceStatus()
  const workspacesQuery = useQuery({
    queryKey: QueryKeys.WORKSPACES,
    queryFn: () => client.workspaces.listAll(),
    select(baseWorkspaces): TWorkspaces {
      // Merge workspaces with existing workspaces status, if we have any
      return baseWorkspaces.map((baseWorkspace) => {
        const maybeStatus = queryClient.getQueryData<TWorkspace>(
          QueryKeys.workspace(baseWorkspace.id)
        )?.status

        if (exists(maybeStatus)) {
          return { ...baseWorkspace, status: maybeStatus }
        }

        return baseWorkspace
      })
    },
    refetchInterval: REFETCH_INTERVAL_MS,
  })

  // Fetching the status for workspaces can take a long time and even time out,
  // so instead of retrieving it together with the workspaces list we
  // regularly check it in the background and update the query cache if it changed
  useQueries({
    queries: (workspacesQuery.data ?? []).map((workspace) => ({
      queryKey: QueryKeys.workspaceStatus(workspace.id),
      queryFn: () => client.workspaces.getStatus(workspace.id),
      onSuccess(newStatus: TWorkspace["status"]) {
        queryClient.setQueryData(QueryKeys.workspace(workspace.id), {
          ...workspace,
          status: newStatus,
        })
        updateWorkspaceStatus(newStatus, { workspaceID: workspace.id })
      },
      refetchInterval: REFETCH_INTERVAL_MS,
      enabled:
        queryClient.isMutating({
          predicate(mutation) {
            // TODO: extract and type
            return mutation.state.variables.workspaceID === workspace.id
          },
        }) === 0,
    })),
  })

  const value = useMemo<TDevpodContext>(
    () => ({
      providers: [
        providersQuery.data,
        { status: providersQuery.status, error: providersQuery.error, target: undefined },
      ],
      workspaces: [
        workspacesQuery.data,
        { status: workspacesQuery.status, error: workspacesQuery.error, target: undefined },
      ],
      updateWorkspaceStatus,
    }),
    [
      providersQuery.data,
      providersQuery.status,
      providersQuery.error,
      workspacesQuery.data,
      workspacesQuery.status,
      workspacesQuery.error,
      updateWorkspaceStatus,
    ]
  )

  return <DevPodContext.Provider value={value}>{children}</DevPodContext.Provider>
}

export function useOngoingOperations() {
  const queryClient = useQueryClient()
  const isMutating = useIsMutating()

  const activeOperations = useMemo<readonly string[]>(() => {
    if (isMutating === 0) {
      return []
    }
    const mutationCache = queryClient.getMutationCache()

    // `TODO: make more efficient :upside_down_face:
    return mutationCache
      .getAll()
      .filter((mutation) => mutation.state.status === "loading")
      .reduce<string[]>((acc, curr) => {
        const maybeMutationKey = curr.options.mutationKey
        const maybeOperationName = maybeMutationKey?.[0] as string

        if (exists(maybeOperationName)) {
          acc.push(maybeOperationName)
        }

        return acc
      }, [])
  }, [queryClient, isMutating])

  return activeOperations
}
