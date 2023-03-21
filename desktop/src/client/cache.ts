import { exists, isEmpty, noop, EventManager, THandler } from "../lib"
import { TUnsubscribeFn, TWorkspaceID } from "../types"
import { TStreamCommandFn, TStreamEvent, TStreamEventListenerFn } from "./commands"

export type TStartCommandHandler = Readonly<{
  promise: Promise<void>
  stream?: (streamHandler?: THandler<TStreamEventListenerFn>) => TUnsubscribeFn
}>
type TStartCommandCacheStore = Map<TWorkspaceID, TStartCommandHandler>
type TStartCommandCache = Pick<TStartCommandCacheStore, "get"> &
  Readonly<{
    connect: (
      workspaceID: TWorkspaceID,
      cmd: Readonly<{ run(): Promise<void>; stream: TStreamCommandFn }>
    ) => Readonly<{
      operation: TStartCommandHandler["promise"]
      stream: TStartCommandHandler["stream"]
    }>
    clear: (workspaceID: TWorkspaceID) => void
  }>

export function createStartCommandCache(): TStartCommandCache {
  const store: TStartCommandCacheStore = new Map()

  return {
    get(id) {
      return store.get(id)
    },
    clear(id) {
      store.delete(id)
    },
    connect(id, cmd) {
      const events: TStreamEvent[] = []
      const eventManager = EventManager.createSingle<TStreamEvent>()

      const promise = cmd.stream((event) => {
        events.push(event)

        eventManager.publish(event)
      })
      const stream: TStartCommandHandler["stream"] = (handler) => {
        if (!exists(handler)) {
          return noop
        }

        // Make sure we subscribe handlers only once
        if (eventManager.isSubscribed(handler)) {
          return () => eventManager.unsubscribe(handler)
        }

        // Replay events in-order before registering the new newHandler
        if (!isEmpty(events)) {
          for (const event of events) {
            handler.notify(event)
          }
        }

        return eventManager.subscribe(handler)
      }

      store.set(id, {
        promise,
        stream,
      })

      return { operation: promise, stream }
    },
  }
}
