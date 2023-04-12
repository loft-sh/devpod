import { TActionName } from "../contexts"
import { exists, isEmpty, noop, ResultError, SingleEventManager, THandler } from "../lib"
import { TUnsubscribeFn } from "../types"
import { TCommand, TStreamEventListenerFn } from "./command"
import { TStreamEvent } from "./types"

export type TCommandCacheInfo = Readonly<{ id: string; actionName: TActionName }>
type TCommandCacheID = `${string}:${TActionName}`
type TCommandHandler = Readonly<{
  promise: Promise<ResultError>
  stream?: (streamHandler?: THandler<TStreamEventListenerFn>) => TUnsubscribeFn
}>
type TCommandCacheStore = Map<TCommandCacheID, TCommandHandler>

export class CommandCache {
  private store: TCommandCacheStore = new Map()

  private getCacheID(info: TCommandCacheInfo): TCommandCacheID {
    return `${info.id}:${info.actionName}`
  }

  public get(info: TCommandCacheInfo): TCommandHandler | undefined {
    const cacheID = this.getCacheID(info)

    return this.store.get(cacheID)
  }

  public clear(info: TCommandCacheInfo) {
    const cacheID = this.getCacheID(info)
    this.store.delete(cacheID)
  }

  public connect<T>(
    info: TCommandCacheInfo,
    cmd: Readonly<TCommand<T>>
  ): Readonly<{
    operation: TCommandHandler["promise"]
    stream: TCommandHandler["stream"]
  }> {
    const events: TStreamEvent[] = []
    const eventManager = new SingleEventManager<TStreamEvent>()

    const promise = cmd.stream((event) => {
      events.push(event)

      eventManager.publish(event)
    })
    const stream: TCommandHandler["stream"] = (handler) => {
      if (!exists(handler)) {
        return noop
      }

      // Replay events in-order before registering the new newHandler
      if (!isEmpty(events)) {
        for (const event of events) {
          handler.notify(event)
        }
      }

      // Make sure we subscribe handlers only once
      if (eventManager.isSubscribed(handler)) {
        return () => eventManager.unsubscribe(handler)
      }

      return eventManager.subscribe(handler)
    }

    const cacheID = this.getCacheID(info)
    this.store.set(cacheID, {
      promise,
      stream,
    })

    return { operation: promise, stream }
  }
}
