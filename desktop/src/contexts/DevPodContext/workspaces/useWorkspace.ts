import { useCallback, useEffect, useId, useMemo, useRef, useSyncExternalStore } from "react"
import { client, TStreamEventListenerFn } from "../../../client"
import { exists } from "../../../lib"
import {
  TDeepNonNullable,
  TUnsubscribeFn,
  TWorkspace,
  TWorkspaceID,
  TWorkspaceStartConfig,
} from "../../../types"
import { TActionID, TPublicAction } from "./action"
import { workspacesStore } from "./workspacesStore"

export type TWorkspaceResult = Readonly<{
  data: TWorkspace | undefined
  isLoading: boolean
  current:
    | (TPublicAction & Readonly<{ connect: (listener: TStreamEventListenerFn) => void }>)
    | undefined
  history: Readonly<{
    // all: readonly TActionObj[]
    replay: (actionID: TActionID, listener: TStreamEventListenerFn) => void
  }>
  start: (config: TWorkspaceStartConfig, onStream?: TStreamEventListenerFn) => void
  create: (
    config: Omit<TWorkspaceStartConfig, "sourceConfig"> &
      Pick<TDeepNonNullable<TWorkspaceStartConfig>, "sourceConfig">,
    onStream?: TStreamEventListenerFn
  ) => void
  stop: (onStream?: TStreamEventListenerFn) => void
  remove: (onStream?: TStreamEventListenerFn) => void
  rebuild: (onStream?: TStreamEventListenerFn) => void
}>

export function useWorkspace(workspaceID: TWorkspaceID | undefined): TWorkspaceResult {
  const viewID = useId()
  const data = useSyncExternalStore(
    useCallback((listener) => workspacesStore.subscribe(listener), []),
    () => (workspaceID !== undefined ? workspacesStore.get(workspaceID) : undefined)
  )
  const create = useCallback<TWorkspaceResult["create"]>(
    async (config, onStream) => {
      workspacesStore.startAction({
        actionName: "create",
        workspaceID: config.id,
        actionFn: async (ctx) => {
          const result = await client.workspaces.start(config, onStream, {
            id: config.id,
            actionID: ctx.id,
            streamID: viewID,
          })
          if (result.err) {
            return result
          }
          workspacesStore.setStatus(config.id, result.val)

          return result
        },
      })
    },
    [viewID]
  )

  const start = useCallback<TWorkspaceResult["start"]>(
    (config, onStream) => {
      if (workspaceID === undefined) {
        return
      }
      workspacesStore.startAction({
        actionName: "start",
        workspaceID,
        actionFn: async (ctx) => {
          const result = await client.workspaces.start(config, onStream, {
            id: workspaceID,
            actionID: ctx.id,
            streamID: viewID,
          })
          if (result.err) {
            return result
          }
          workspacesStore.setStatus(workspaceID, result.val)

          return result
        },
      })
    },
    [viewID, workspaceID]
  )

  const stop = useCallback<TWorkspaceResult["stop"]>(
    (onStream) => {
      if (workspaceID === undefined) {
        return
      }
      workspacesStore.startAction({
        actionName: "stop",
        workspaceID,
        actionFn: async (ctx) => {
          const result = await client.workspaces.stop(onStream, {
            id: workspaceID,
            actionID: ctx.id,
            streamID: viewID,
          })
          if (result.err) {
            return result
          }
          workspacesStore.setStatus(workspaceID, result.val)

          return result
        },
      })
    },
    [viewID, workspaceID]
  )

  const rebuild = useCallback<TWorkspaceResult["rebuild"]>(
    (onStream) => {
      if (workspaceID === undefined) {
        return
      }
      workspacesStore.startAction({
        actionName: "rebuild",
        workspaceID,
        actionFn: async (ctx) => {
          const result = await client.workspaces.rebuild(onStream, {
            id: workspaceID,
            actionID: ctx.id,
            streamID: viewID,
          })
          if (result.err) {
            return result
          }
          workspacesStore.setStatus(workspaceID, result.val)

          return result
        },
      })
    },
    [viewID, workspaceID]
  )

  const remove = useCallback<TWorkspaceResult["remove"]>(
    (onStream) => {
      if (workspaceID === undefined) {
        return
      }
      workspacesStore.startAction({
        actionName: "remove",
        workspaceID,
        actionFn: async (ctx) => {
          const result = await client.workspaces.remove(onStream, {
            id: workspaceID,
            actionID: ctx.id,
            streamID: viewID,
          })
          if (result.err) {
            return result
          }
          workspacesStore.removeWorkspace(workspaceID)

          return result
        },
      })
    },
    [viewID, workspaceID]
  )

  const currentAction = useSyncExternalStore(
    useCallback((listener) => workspacesStore.subscribe(listener), []),
    () => (workspaceID !== undefined ? workspacesStore.getCurrentAction(workspaceID) : undefined)
  )
  const isLoading = useMemo(() => exists(currentAction), [currentAction])

  const subscriptionRef = useRef<TUnsubscribeFn>()
  // Make sure we unsubscribe on onmount
  useEffect(() => {
    return () => subscriptionRef.current?.()
  }, [])

  // Unsubscribe whenever action changes
  useEffect(() => {
    if (
      (currentAction === undefined || currentAction.status !== "pending") &&
      subscriptionRef.current !== undefined
    ) {
      subscriptionRef.current()
      subscriptionRef.current = undefined
    }
  }, [currentAction])

  const current = useMemo<TWorkspaceResult["current"]>(() => {
    if (currentAction === undefined) {
      return undefined
    }

    return {
      ...currentAction,
      connect: (onStream: TStreamEventListenerFn) => {
        subscriptionRef.current = client.workspaces.subscribe(currentAction, viewID, onStream)
      },
    }
  }, [currentAction, viewID])

  const history = useMemo<TWorkspaceResult["history"]>(() => {
    return {
      replay: (actionID, listener) => {
        client.workspaces.replayAction(actionID, listener)
      },
    }
  }, [])

  return useMemo(
    () => ({
      data,
      isLoading,
      current,
      history,
      create,
      start,
      stop,
      rebuild,
      remove,
    }),
    [data, isLoading, current, history, create, start, stop, rebuild, remove]
  )
}
