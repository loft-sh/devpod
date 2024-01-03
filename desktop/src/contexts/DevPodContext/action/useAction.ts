import { useCallback, useEffect, useId, useMemo, useRef, useSyncExternalStore } from "react"
import { client, TStreamEventListenerFn } from "../../../client"
import { TStreamID, TUnsubscribeFn } from "../../../types"
import { devPodStore } from "../devPodStore"
import { TActionID, TActionObj } from "./action"

type TActionResult = Readonly<{
  data: TActionObj
  connectOrReplay(onStream: TStreamEventListenerFn): void | VoidFunction
  cancel(): void
}>

export function useAction(actionID: TActionID | undefined): TActionResult | undefined {
  const isCancellingRef = useRef(false)
  const viewID = useId()
  const data = useSyncExternalStore(
    useCallback((listener) => devPodStore.subscribe(listener), []),
    () => {
      if (actionID === undefined) {
        return undefined
      }

      return getAction(actionID)
    }
  )

  const connect = useConnectAction(data, viewID)
  const replay = useReplayAction()

  return useMemo(() => {
    if (data === undefined) {
      return undefined
    }

    return {
      data,
      connectOrReplay: (onStream) => {
        if (data.status === "pending") {
          return connect(onStream)
        }

        return replay(data.id, onStream)
      },
      cancel: () => {
        if (isCancellingRef.current) {
          return
        }
        isCancellingRef.current = true
        // could improve by setting timeout as fallback if promise doesn't resolve, let's see if this is enough
        client.workspaces.cancelAction(data.targetID).finally(() => {
          isCancellingRef.current = false
        })
      },
    }
  }, [data, connect, replay])
}

export function getAction(actionID: TActionID): TActionObj | undefined {
  const { active, history } = devPodStore.getAllActions()

  return [...active, ...history].find((action) => action.id === actionID)
}

export function useConnectAction(
  action: TActionObj | undefined,
  streamID: TStreamID
): (onStream: TStreamEventListenerFn) => void {
  const subscriptionRef = useRef<TUnsubscribeFn>()
  // Make sure we unsubscribe on onmount
  useEffect(() => {
    return () => subscriptionRef.current?.()
  }, [])

  // Unsubscribe whenever action changes
  useEffect(() => {
    if (
      (action === undefined || action.status !== "pending") &&
      subscriptionRef.current !== undefined
    ) {
      subscriptionRef.current()
      subscriptionRef.current = undefined
    }
  }, [action])

  return useCallback(
    (onStream) => {
      if (action === undefined) {
        return
      }

      subscriptionRef.current = client.workspaces.subscribe(action, streamID, onStream)
    },
    [action, streamID]
  )
}

export function useReplayAction(): (
  actionID: TActionID,
  onStream: TStreamEventListenerFn
) => VoidFunction {
  return useCallback((actionID, onStream) => client.workspaces.replayAction(actionID, onStream), [])
}
