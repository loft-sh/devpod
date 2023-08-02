import { TComparable, TIdentifiable, TUnsubscribeFn } from "../types"
import { exists, isEmpty } from "./helpers"
import { v4 as uuidv4 } from "uuid"

type TEventHandler<TEvents, TEventName extends keyof TEvents = keyof TEvents> = THandler<
  (event: TEvents[TEventName]) => void
>
export type THandler<TFn = Function> = Readonly<{ notify: TFn }> &
  TIdentifiable &
  TComparable<TIdentifiable>
type TBaseEvents = Record<string | number | symbol, unknown>

type TEventManager<TEvents extends TBaseEvents> = Readonly<{
  subscribe: <TEventName extends keyof TEvents>(
    eventName: TEventName,
    handler: TEventHandler<TEvents, TEventName>
  ) => TUnsubscribeFn
  isSubscribed: <TEventName extends keyof TEvents>(
    eventName: TEventName,
    handler: TEventHandler<TEvents, TEventName>
  ) => boolean
  unsubscribe: <TEventName extends keyof TEvents>(
    eventName: TEventName,
    handler: TEventHandler<TEvents, TEventName>
  ) => void
  publish: <TEventName extends keyof TEvents>(
    eventName: TEventName,
    event: TEvents[TEventName]
  ) => boolean
  clear: <TEventName extends keyof TEvents>(eventName: TEventName) => void
}>

export class EventManager<TEvents extends TBaseEvents> implements TEventManager<TEvents> {
  private handlers = new Map<keyof TEvents, TEventHandler<TEvents>[]>()

  public static toHandler<TFn extends Function>(
    listenerFn: TFn,
    id: string = uuidv4()
  ): THandler<TFn> {
    return {
      id,
      eq(other) {
        return this.id === other.id
      },
      notify: listenerFn,
    }
  }

  public subscribe<TEventName extends keyof TEvents>(
    eventName: TEventName,
    handler: TEventHandler<TEvents, TEventName>
  ): VoidFunction {
    const maybeHandlers = this.handlers.get(eventName)
    if (!exists(maybeHandlers)) {
      this.handlers.set(eventName, [handler as TEventHandler<TEvents>])
    } else {
      this.handlers.set(eventName, [...maybeHandlers, handler as TEventHandler<TEvents>])
    }

    return () => this.unsubscribe(eventName, handler)
  }

  public isSubscribed<TEventName extends keyof TEvents>(
    eventName: TEventName,
    handler: TEventHandler<TEvents, TEventName>
  ): boolean {
    const maybeHandlers = this.handlers.get(eventName)
    if (!exists(maybeHandlers)) {
      return false
    }

    return exists(maybeHandlers.find((l) => l.eq(handler)))
  }

  public unsubscribe<TEventName extends keyof TEvents>(
    eventName: TEventName,
    handler: TEventHandler<TEvents, TEventName>
  ): void {
    const maybeEventHandlers = this.handlers.get(eventName)
    if (!exists(maybeEventHandlers)) {
      return
    }

    this.handlers.set(
      eventName,
      maybeEventHandlers.filter((h) => !h.eq(handler))
    )

    if (isEmpty(maybeEventHandlers)) {
      this.handlers.delete(eventName)
    }
  }

  public publish<TEventName extends keyof TEvents>(
    eventName: TEventName,
    event: TEvents[TEventName]
  ): boolean {
    const maybeHandlers = this.handlers.get(eventName)
    if (!exists(maybeHandlers)) {
      return false
    }

    for (const handler of maybeHandlers) {
      handler.notify(event)
    }

    return true
  }

  public clear<TEventName extends keyof TEvents>(eventName: TEventName): void {
    this.handlers.delete(eventName)
  }
}

export class SingleEventManager<T> {
  private manager = new EventManager<{ event: T }>()

  subscribe(handler: TEventHandler<{ event: T }, "event">): VoidFunction {
    return this.manager.subscribe("event", handler)
  }

  isSubscribed(handler: TEventHandler<{ event: T }, "event">): boolean {
    return this.manager.isSubscribed("event", handler)
  }

  unsubscribe(handler: TEventHandler<{ event: T }, "event">): void {
    return this.manager.unsubscribe("event", handler)
  }

  publish(event: T): boolean {
    return this.manager.publish("event", event)
  }

  clear(): void {
    return this.manager.clear("event")
  }
}
