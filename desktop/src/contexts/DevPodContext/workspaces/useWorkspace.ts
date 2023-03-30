import { useCallback, useMemo, useSyncExternalStore } from "react"
import { client, TStreamEventListenerFn } from "../../../client"
import { exists, Result } from "../../../lib"
import { TDeepNonNullable, TWorkspace, TWorkspaceID, TWorkspaceStartConfig } from "../../../types"
import { TPublicAction } from "./action"
import { workspacesStore } from "./workspacesStore"

export type TWorkspaceResult = Readonly<{
  data: TWorkspace | undefined
  isLoading: boolean
  current: TPublicAction | undefined
  // history: Readonly<{
  //   get: (actionID: TActionID) => TAction
  //   getAll: (actionName: TActionName) => readonly TAction[]
  // }>
  start: (config: TWorkspaceStartConfig, onStream?: TStreamEventListenerFn) => void
  create: (
    config: Omit<TWorkspaceStartConfig, "sourceConfig"> &
      Pick<TDeepNonNullable<TWorkspaceStartConfig>, "sourceConfig">,
    onStream?: TStreamEventListenerFn
  ) => Promise<Result<TWorkspaceID>>
  stop: VoidFunction
  remove: VoidFunction
  rebuild: VoidFunction
}>

export function useWorkspace(workspaceID: TWorkspaceID | undefined): TWorkspaceResult {
  const workspace = useSyncExternalStore(
    useCallback((listener) => workspacesStore.subscribe(listener), []),
    () => (workspaceID !== undefined ? workspacesStore.get(workspaceID) : undefined)
  )
  const currentAction = useSyncExternalStore(
    useCallback((listener) => workspacesStore.subscribe(listener), []),
    () => (workspaceID !== undefined ? workspacesStore.getCurrentAction(workspaceID) : undefined)
  )
  const isLoading = useMemo(() => exists(currentAction), [currentAction])

  const create = useCallback<TWorkspaceResult["create"]>(async (config, onStream) => {
    const newIDResult = await client.workspaces.newID(config.sourceConfig.source)
    if (newIDResult.err) {
      return newIDResult
    }
    const workspaceID = newIDResult.val

    workspacesStore.startAction({
      actionName: "create",
      workspaceID,
      actionFn: async (ctx) => {
        const result = await client.workspaces.start(workspaceID, config, ctx.id, onStream)
        if (result.err) {
          return result
        }
        workspacesStore.setStatus(workspaceID, result.val)

        return result
      },
    })

    return newIDResult
  }, [])

  const start = useCallback<TWorkspaceResult["start"]>(
    (config, onStream) => {
      if (workspaceID === undefined) {
        return
      }
      workspacesStore.startAction({
        actionName: "start",
        workspaceID,
        actionFn: async (ctx) => {
          const result = await client.workspaces.start(workspaceID, config, ctx.id, onStream)
          if (result.err) {
            return result
          }
          workspacesStore.setStatus(workspaceID, result.val)

          return result
        },
      })
    },
    [workspaceID]
  )

  const stop = useCallback<TWorkspaceResult["stop"]>(() => {
    if (workspaceID === undefined) {
      return
    }
    workspacesStore.startAction({
      actionName: "stop",
      workspaceID,
      actionFn: async () => {
        const result = await client.workspaces.stop(workspaceID)
        if (result.err) {
          return result
        }
        workspacesStore.setStatus(workspaceID, result.val)

        return result
      },
    })
  }, [workspaceID])

  const rebuild = useCallback<TWorkspaceResult["rebuild"]>(() => {
    if (workspaceID === undefined) {
      return
    }
    workspacesStore.startAction({
      actionName: "rebuild",
      workspaceID,
      actionFn: async () => {
        const result = await client.workspaces.rebuild(workspaceID)
        if (result.err) {
          return result
        }
        workspacesStore.setStatus(workspaceID, result.val)

        return result
      },
    })
  }, [workspaceID])

  const remove = useCallback<TWorkspaceResult["remove"]>(() => {
    if (workspaceID === undefined) {
      return
    }
    workspacesStore.startAction({
      actionName: "remove",
      workspaceID,
      actionFn: async () => {
        const result = await client.workspaces.remove(workspaceID)
        if (result.err) {
          return result
        }
        workspacesStore.setStatus(workspaceID, result.val)

        return result
      },
    })
  }, [workspaceID])

  return useMemo(
    () => ({
      data: workspace,
      isLoading,
      current: currentAction,
      create,
      start,
      stop,
      rebuild,
      remove,
    }),
    [workspace, isLoading, currentAction, create, start, stop, rebuild, remove]
  )
}
