import { exists, isEmpty, noop, SingleEventManager, THandler } from "../lib"
import { TUnsubscribeFn, TWorkspaceID } from "../types"
import {ResultError} from "../lib/result";
import {TCommand, TStreamEvent, TStreamEventListenerFn} from "./command";
import {ChildProcess} from "@tauri-apps/api/shell";

export type TStartCommandHandler = Readonly<{
  promise: Promise<ResultError>
  stream?: (streamHandler?: THandler<TStreamEventListenerFn>) => TUnsubscribeFn
}>
type TStartCommandCacheStore = Map<TWorkspaceID, TStartCommandHandler>
type TStartCommandCache = Pick<TStartCommandCacheStore, "get"> &
  Readonly<{
    connect: (
      workspaceID: TWorkspaceID,
      cmd: Readonly<TCommand<ChildProcess>>
    ) => Readonly<{
      operation: TStartCommandHandler["promise"]
      stream: TStartCommandHandler["stream"]
    }>
    clear: (workspaceID: TWorkspaceID) => void
  }>

export class StartCommandCache implements TStartCommandCache {
  private store: TStartCommandCacheStore = new Map()

  public get(id: TWorkspaceID) {
    return this.store.get(id)
  }

  public clear(id: TWorkspaceID) {
    this.store.delete(id)
  }

  public connect(
    id: TWorkspaceID,
    cmd: Readonly<TCommand<ChildProcess>>
  ): Readonly<{
    operation: TStartCommandHandler["promise"]
    stream: TStartCommandHandler["stream"]
  }> {
    const events: TStreamEvent[] = []
    const eventManager = new SingleEventManager<TStreamEvent>()

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

    this.store.set(id, {
      promise,
      stream,
    })

    return { operation: promise, stream }
  }
}
