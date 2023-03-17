import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query"
import { createContext, ReactNode, useCallback, useContext, useMemo } from "react"
import { client } from "../../client"
import { exists } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import {
  TProviders,
  TQueryResult,
  TWithWorkspaceID,
  TWorkspace,
  TWorkspaceID,
  TWorkspaceManager,
  TWorkspaceManagerRunConfig,
  TWorkspaces,
} from "../../types"
import { getOperationManagerFromMutation, mergeWorkspaceStatus } from "./helpers"

type TDevpodContext = Readonly<{
  providers: TQueryResult<TProviders>
  workspaces: TQueryResult<TWorkspaces>
  updateWorkspaceStatus: (newStatus: TWorkspace["status"], config: TWithWorkspaceID) => void
}>
const DevPodContext = createContext<TDevpodContext>(null!)

const REFETCH_INTERVAL_MS = 1_000

export function DevPodProvider({ children }: Readonly<{ children?: ReactNode }>) {
  const queryClient = useQueryClient()
  const providersQuery = useQuery([QueryKeys.PROVIDERS], () => undefined, {
    refetchInterval: REFETCH_INTERVAL_MS,
    enabled: false, // FIXME: enable if ready
  })

  const updateWorkspaceStatus = useCallback<TDevpodContext["updateWorkspaceStatus"]>(
    (newStatus, { workspaceID }) => {
      // update cache with new status
      queryClient.setQueryData<TWorkspaces>(
        [QueryKeys.WORKSPACES],
        mergeWorkspaceStatus({ id: workspaceID, status: newStatus })
      )

      const workspaceKey = QueryKeys.workspace(workspaceID)
      queryClient.setQueryData<TWorkspace>(workspaceKey, (currentWorkspace) =>
        exists(currentWorkspace) ? { ...currentWorkspace, status: newStatus } : currentWorkspace
      )
    },
    [queryClient]
  )

  const workspacesQuery = useQuery([QueryKeys.WORKSPACES], () => client.workspaces.listAll(), {
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
    })),
  })

  const value = useMemo<TDevpodContext>(
    () => ({
      providers: [
        providersQuery.data,
        { status: providersQuery.status, error: providersQuery.error },
      ],
      workspaces: [
        workspacesQuery.data,
        { status: workspacesQuery.status, error: workspacesQuery.error },
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

export function useWorkspaces(): [TDevpodContext["workspaces"], TWorkspaceManager] {
  const { workspaces } = useContext(DevPodContext)
  const manager = useWorkspaceManager()

  return useMemo(() => [workspaces, manager], [manager, workspaces])
}

export function useWorkspaceManager(): TWorkspaceManager {
  const { updateWorkspaceStatus } = useContext(DevPodContext)
  const queryClient = useQueryClient()

  const createMutation = useMutation({
    mutationFn: async ({
      rawWorkspaceSource,
      config,
      onStream,
    }: TWorkspaceManagerRunConfig<"create">) => {
      // At this point we don't have a workspaceID yet, so we need to create one
      const workspaceID = await client.workspaces.newWorkspaceID(rawWorkspaceSource)
      const status = await client.workspaces.start(workspaceID, config, onStream)

      return { status, workspaceID }
    },
    onSuccess({ status, workspaceID }) {
      updateWorkspaceStatus(status, { workspaceID })
    },
  })
  const startMutation = useMutation({
    mutationFn: ({ workspaceID, config, onStream }: TWorkspaceManagerRunConfig<"start">) =>
      client.workspaces.start(workspaceID, config, onStream),
    onSuccess: updateWorkspaceStatus,
  })
  const stopMutation = useMutation({
    mutationFn: ({ workspaceID }: TWorkspaceManagerRunConfig<"stop">) =>
      client.workspaces.stop(workspaceID),
    onSuccess: updateWorkspaceStatus,
  })
  const rebuildMutation = useMutation({
    mutationFn: ({ workspaceID }: TWorkspaceManagerRunConfig<"rebuild">) =>
      client.workspaces.rebuild(workspaceID),
    onSuccess: updateWorkspaceStatus,
  })
  const removeMutation = useMutation({
    mutationFn: async ({ workspaceID }: TWorkspaceManagerRunConfig<"remove">) => {
      await client.workspaces.remove(workspaceID)

      return Promise.resolve()
    },
    onSuccess(_, { workspaceID }) {
      queryClient.setQueryData<TWorkspaces>([QueryKeys.WORKSPACES], (currentWorkspaces) =>
        currentWorkspaces?.filter((workspace) => workspace.id !== workspaceID)
      )
    },
  })

  return useMemo(
    () => ({
      create: {
        run: createMutation.mutate,
        status: createMutation.status,
        error: createMutation.error,
      },
      start: getOperationManagerFromMutation<
        TWorkspaceManagerRunConfig<"start">, // let's help typescript out a bit here
        typeof startMutation
      >(startMutation),
      stop: getOperationManagerFromMutation(stopMutation),
      remove: getOperationManagerFromMutation(removeMutation),
      rebuild: getOperationManagerFromMutation(rebuildMutation),
    }),
    [
      createMutation.error,
      createMutation.mutate,
      createMutation.status,
      rebuildMutation,
      removeMutation,
      startMutation,
      stopMutation,
    ]
  )
}

export function useWorkspace(
  workspaceID: TWorkspaceID | undefined
): [TQueryResult<TWorkspace>, TWorkspaceManager] {
  const queryClient = useQueryClient()
  const manager = useWorkspaceManager()

  const { data, status, error } = useQuery(
    QueryKeys.workspace(workspaceID!), // force non-null because of `enabled` query config
    ({ queryKey }) => {
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
    { enabled: exists(workspaceID) }
  )

  return [[data, { status, error }], manager]
}

export function useProviders(): TDevpodContext["providers"] {
  return useContext(DevPodContext).providers
}
