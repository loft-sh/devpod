import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCallback, useEffect, useId, useMemo, useRef } from "react"
import { client } from "../../client"
import { MutationKeys, QueryKeys } from "../../queryKeys"
import {
  TConnectOperationFn,
  TWithWorkspaceID,
  TWorkspaceManager,
  TWorkspaceManagerRunConfig,
  TWorkspaces,
} from "../../types"
import { getOperationManagerFromMutation } from "./helpers"
import { useUpdateWorkspaceStatus } from "./useUpdateWorkspaceStatus"

export function useWorkspaceManager(): TWorkspaceManager {
  const viewID = useId()
  const startSubscriptionRef = useRef<VoidFunction>()
  const updateWorkspaceStatus = useUpdateWorkspaceStatus()
  const queryClient = useQueryClient()

  const createMutation = useMutation({
    mutationKey: MutationKeys.CREATE,
    mutationFn: async ({
      rawWorkspaceSource,
      config,
      onStream,
    }: TWorkspaceManagerRunConfig["create"]) => {
      // At this point we don't have a workspaceID yet, so we need to create one
      const workspaceID = await client.workspaces.newWorkspaceID(rawWorkspaceSource)
      const status = await client.workspaces.start(workspaceID, config, viewID, onStream)

      return { status, workspaceID }
    },
    onSuccess({ status, workspaceID }) {
      updateWorkspaceStatus(status, { workspaceID })
    },
  })
  const startMutation = useMutation({
    mutationKey: MutationKeys.START,
    mutationFn: ({ workspaceID, config, onStream }: TWorkspaceManagerRunConfig["start"]) =>
      client.workspaces.start(workspaceID, config, viewID, onStream),
    onSuccess: updateWorkspaceStatus,
  })
  const stopMutation = useMutation({
    mutationKey: MutationKeys.STOP,
    mutationFn: ({ workspaceID }: TWorkspaceManagerRunConfig["stop"]) =>
      client.workspaces.stop(workspaceID),
    onSuccess: updateWorkspaceStatus,
  })
  const rebuildMutation = useMutation({
    mutationKey: MutationKeys.REBUILD,
    mutationFn: ({ workspaceID }: TWorkspaceManagerRunConfig["rebuild"]) =>
      client.workspaces.rebuild(workspaceID),
    onSuccess: updateWorkspaceStatus,
  })
  const removeMutation = useMutation({
    mutationKey: MutationKeys.REMOVE,
    mutationFn: async ({ workspaceID }: TWorkspaceManagerRunConfig["remove"]) => {
      await client.workspaces.remove(workspaceID)

      return Promise.resolve()
    },
    onSuccess(_, { workspaceID }) {
      queryClient.setQueryData<TWorkspaces>(QueryKeys.WORKSPACES, (currentWorkspaces) =>
        currentWorkspaces?.filter((workspace) => workspace.id !== workspaceID)
      )
    },
  })

  const connectStart = useCallback<TConnectOperationFn<TWithWorkspaceID>>(
    (config) => {
      startSubscriptionRef.current = client.workspaces.subscribeToStart(
        config.workspaceID,
        viewID,
        config.onStream
      )
    },
    [viewID]
  )

  // Unsubscribe on unmount
  useEffect(() => {
    return () => startSubscriptionRef.current?.()
  }, [])

  return useMemo(
    () => ({
      create: {
        run: createMutation.mutate,
        status: createMutation.status,
        error: createMutation.error,
        target: createMutation.variables,
      },
      start: {
        ...getOperationManagerFromMutation(startMutation),
        connect: connectStart,
      },
      stop: getOperationManagerFromMutation(stopMutation),
      remove: getOperationManagerFromMutation(removeMutation),
      rebuild: getOperationManagerFromMutation(rebuildMutation),
    }),
    [
      connectStart,
      createMutation.error,
      createMutation.mutate,
      createMutation.status,
      createMutation.variables,
      rebuildMutation,
      removeMutation,
      startMutation,
      stopMutation,
    ]
  )
}
