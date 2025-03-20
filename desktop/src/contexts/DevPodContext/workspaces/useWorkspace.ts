import { useCallback, useId, useMemo, useRef, useSyncExternalStore } from "react"
import { TStreamEventListenerFn, client } from "../../../client"
import { exists } from "../../../lib"
import { TIdentifiable, TStreamID, TWorkspaceID, TWorkspaceStartConfig } from "../../../types"
import { TActionID, TActionObj, useConnectAction, useReplayAction } from "../action"
import { IWorkspaceStore, ProWorkspaceStore, useWorkspaceStore } from "../workspaceStore"

export type TWorkspaceResult<T> = Readonly<{
  data: T | undefined
  isLoading: boolean
  current:
    | (TActionObj & Readonly<{ connect: (listener: TStreamEventListenerFn) => void }>)
    | undefined
  history: Readonly<{
    // all: readonly TActionObj[]
    replay: (actionID: TActionID, listener: TStreamEventListenerFn) => void
  }>
  start: (config: TWorkspaceStartConfig, onStream?: TStreamEventListenerFn) => TActionID | undefined
  create: (
    config: Omit<TWorkspaceStartConfig, "sourceConfig"> &
      Pick<TWorkspaceStartConfig, "sourceConfig"> &
      Readonly<{ workspaceKey?: string }>,
    onStream?: TStreamEventListenerFn
  ) => TActionID
  stop: (onStream?: TStreamEventListenerFn) => TActionID | undefined
  remove: (force: boolean, onStream?: TStreamEventListenerFn) => TActionID | undefined
  rebuild: (onStream?: TStreamEventListenerFn) => TActionID | undefined
  reset: (onStream?: TStreamEventListenerFn) => TActionID | undefined
  checkStatus: (onStream?: TStreamEventListenerFn) => TActionID | undefined
}>

export function useWorkspaceActions(
  workspaceID: TWorkspaceID | undefined
): TActionObj[] | undefined {
  const { store } = useWorkspaceStore()
  const dataCache = useRef<TActionObj[]>()
  const data = useSyncExternalStore(
    useCallback((listener) => store.subscribe(listener), [store]),
    () => {
      if (workspaceID === undefined) {
        return undefined
      }

      // It's okay to use sort directly here because the store always returns a new array
      const workspaceActions = store.getWorkspaceActions(workspaceID).sort((a, b) => {
        if (a.finishedAt && b.finishedAt) {
          return b.finishedAt - a.finishedAt
        }

        return b.createdAt - a.createdAt
      })
      if (!dataCache.current || dataCache.current.length !== workspaceActions.length) {
        dataCache.current = workspaceActions

        return dataCache.current
      }

      // compare actions
      const diff = dataCache.current.filter(
        (action) => !workspaceActions.find((workspaceAction) => action.id === workspaceAction.id)
      )
      if (diff.length > 0) {
        dataCache.current = workspaceActions

        return dataCache.current
      }

      return dataCache.current
    }
  )

  return data
}

export function useWorkspace<TW extends TIdentifiable>(
  workspaceKey: string | undefined
): TWorkspaceResult<TW> {
  const { store } = useWorkspaceStore<IWorkspaceStore<string, TW>>()
  const viewID = useId()
  const data = useSyncExternalStore(
    useCallback((listener) => store.subscribe(listener), [store]),
    () => (workspaceKey !== undefined ? store.get(workspaceKey) : undefined)
  )
  const workspaceID = useMemo(() => {
    if (!data) {
      return undefined
    }

    return data.id
  }, [data])

  const create = useCallback<TWorkspaceResult<TW>["create"]>(
    (config, onStream) => {
      return store.startAction({
        actionName: "start",
        workspaceKey: config.workspaceKey ?? config.id,
        actionFn: async (ctx) => {
          const result = await client.workspaces.start(config, onStream, {
            id: config.id,
            actionID: ctx.id,
            streamID: viewID,
          })
          if (result.err) {
            return result
          }
          store.setStatus(config.id, result.val)

          return result
        },
      })
    },
    [store, viewID]
  )

  const start = useCallback<TWorkspaceResult<TW>["start"]>(
    (config, onStream) => {
      if (workspaceID === undefined) {
        return
      }

      return startWorkspaceAction({ workspaceID, config, onStream, streamID: viewID, store })
    },
    [store, viewID, workspaceID]
  )

  const checkStatus = useCallback<TWorkspaceResult<TW>["checkStatus"]>(
    (onStream) => {
      if (workspaceID === undefined) {
        return
      }

      return store.startAction({
        actionName: "checkStatus",
        workspaceKey: workspaceID,
        actionFn: async (ctx) => {
          const result = await client.workspaces.checkStatus(onStream, {
            id: workspaceID,
            actionID: ctx.id,
            streamID: viewID,
          })
          if (result.err) {
            return result
          }
          store.setStatus(workspaceID, result.val)

          return result
        },
      })
    },
    [store, viewID, workspaceID]
  )

  const stop = useCallback<TWorkspaceResult<TW>["stop"]>(
    (onStream) => {
      if (workspaceID === undefined) {
        return
      }

      return stopWorkspaceAction({ workspaceID, onStream, streamID: viewID, store })
    },
    [store, viewID, workspaceID]
  )

  const rebuild = useCallback<TWorkspaceResult<TW>["rebuild"]>(
    (onStream) => {
      if (workspaceID === undefined) {
        return
      }

      return store.startAction({
        actionName: "rebuild",
        workspaceKey: workspaceID,
        actionFn: async (ctx) => {
          const result = await client.workspaces.rebuild(onStream, {
            id: workspaceID,
            actionID: ctx.id,
            streamID: viewID,
          })
          if (result.err) {
            return result
          }
          store.setStatus(workspaceID, result.val)

          return result
        },
      })
    },
    [store, viewID, workspaceID]
  )

  const reset = useCallback<TWorkspaceResult<TW>["reset"]>(
    (onStream) => {
      if (workspaceID === undefined) {
        return
      }

      return store.startAction({
        actionName: "reset",
        workspaceKey: workspaceID,
        actionFn: async (ctx) => {
          const result = await client.workspaces.reset(onStream, {
            id: workspaceID,
            actionID: ctx.id,
            streamID: viewID,
          })
          if (result.err) {
            return result
          }
          store.setStatus(workspaceID, result.val)

          return result
        },
      })
    },
    [store, viewID, workspaceID]
  )

  const remove = useCallback<TWorkspaceResult<TW>["remove"]>(
    (force, onStream) => {
      if (workspaceID === undefined) {
        return
      }

      return removeWorkspaceAction({ force, workspaceID, onStream, streamID: viewID, store })
    },
    [store, viewID, workspaceID]
  )

  const currentAction = useSyncExternalStore(
    useCallback((listener) => store.subscribe(listener), [store]),
    () => (workspaceID !== undefined ? store.getCurrentAction(workspaceID) : undefined)
  )
  const isLoading = useMemo(() => exists(currentAction), [currentAction])

  const connect = useConnectAction(currentAction, viewID)
  const current = useMemo<TWorkspaceResult<TW>["current"]>(() => {
    if (currentAction === undefined) {
      return undefined
    }

    return {
      ...currentAction,
      connect,
    }
  }, [currentAction, connect])

  const replay = useReplayAction()
  const history = useMemo<TWorkspaceResult<TW>["history"]>(() => {
    return { replay }
  }, [replay])

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
      reset,
      remove,
      checkStatus,
    }),
    [data, isLoading, current, history, create, start, stop, rebuild, reset, remove, checkStatus]
  )
}

type TStartWorkspaceActionArgs = Readonly<{
  config: TWorkspaceStartConfig
  onStream?: TStreamEventListenerFn
  workspaceID: TWorkspaceID
  streamID: TStreamID
  store: IWorkspaceStore<string, unknown>
}>
export function startWorkspaceAction({
  workspaceID,
  streamID,
  config,
  onStream,
  store,
}: TStartWorkspaceActionArgs): TActionObj["id"] {
  return store.startAction({
    actionName: "start",
    workspaceKey: workspaceID,
    actionFn: async (ctx) => {
      const result = await client.workspaces.start(config, onStream, {
        id: workspaceID,
        actionID: ctx.id,
        streamID,
      })
      if (result.err) {
        return result
      }
      store.setStatus(workspaceID, result.val)

      return result
    },
  })
}

type TStopWorkspaceActionArgs = Readonly<{
  onStream?: TStreamEventListenerFn
  workspaceID: TWorkspaceID
  streamID: TStreamID
  store: IWorkspaceStore<string, unknown>
}>
export function stopWorkspaceAction({
  workspaceID,
  onStream,
  streamID,
  store,
}: TStopWorkspaceActionArgs): TActionObj["id"] {
  return store.startAction({
    actionName: "stop",
    workspaceKey: workspaceID,
    actionFn: async (ctx) => {
      const result = await client.workspaces.stop(onStream, {
        id: workspaceID,
        actionID: ctx.id,
        streamID,
      })
      if (result.err) {
        return result
      }
      store.setStatus(workspaceID, result.val)

      return result
    },
  })
}

type TRemoveWorkspaceActionArgs = Readonly<{
  onStream?: TStreamEventListenerFn
  workspaceID: TWorkspaceID
  streamID: TStreamID
  force: boolean
  store: IWorkspaceStore<string, unknown>
}>
export function removeWorkspaceAction({
  workspaceID,
  onStream,
  streamID,
  force,
  store,
}: TRemoveWorkspaceActionArgs): TActionObj["id"] {
  return store.startAction({
    actionName: "remove",
    workspaceKey: workspaceID,
    actionFn: async (ctx) => {
      const result = await client.workspaces.remove(force, onStream, {
        id: workspaceID,
        actionID: ctx.id,
        streamID,
      })
      if (result.err) {
        return result
      }
      // Pro Desktop app will get updates through watcher, no need to remove from local store
      if (store instanceof ProWorkspaceStore) {
        return result
      }

      store.removeWorkspace(workspaceID)

      return result
    },
  })
}
