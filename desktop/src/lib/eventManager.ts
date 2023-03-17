import { TComparable, TIdentifiable, TUnsubscribeFn } from "../types"
import { exists, isEmpty } from "./helpers"

type TEventHandler<
  TEvents extends TBaseEvents,
  TEventName extends keyof TEvents = keyof TEvents
> = THandler<(event: TEvents[TEventName]) => void>
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

export const EventManager = {
  create: createEventManager,
  createSingle: createSingleEventManager,
  toHandler: createHandler,
}

function createHandler<TFn extends Function>(
  listenerFn: TFn,
  id: string = crypto.randomUUID()
): THandler<TFn> {
  return {
    id,
    eq(other) {
      return this.id === other.id
    },
    notify: listenerFn,
  }
}

function createEventManager<TEvents extends TBaseEvents>(): TEventManager<TEvents> {
  const handlers = new Map<keyof TEvents, TEventHandler<TEvents>[]>()

  return {
    subscribe(eventName, handler) {
      const maybeHandlers = handlers.get(eventName)
      if (!exists(maybeHandlers)) {
        handlers.set(eventName, [handler as TEventHandler<TEvents>])
      } else {
        handlers.set(eventName, [...maybeHandlers, handler as TEventHandler<TEvents>])
      }

      return () => this.unsubscribe(eventName, handler)
    },
    isSubscribed(eventName, handler) {
      const maybeHandlers = handlers.get(eventName)
      if (!exists(maybeHandlers)) {
        return false
      }

      return exists(maybeHandlers.find((l) => l.eq(handler)))
    },
    unsubscribe(eventName, handler) {
      const maybeEventHandlers = handlers.get(eventName)
      if (!exists(maybeEventHandlers)) {
        return
      }

      handlers.set(
        eventName,
        maybeEventHandlers.filter((h) => !h.eq(handler))
      )

      if (isEmpty(maybeEventHandlers)) {
        handlers.delete(eventName)
      }
    },
    publish(eventName, event): boolean {
      const maybeHandlers = handlers.get(eventName)
      if (!exists(maybeHandlers)) {
        return false
      }

      for (const handler of maybeHandlers) {
        handler.notify(event)
      }

      return true
    },
    clear(eventName) {
      handlers.delete(eventName)
    },
  }
}

// Mapped helper type, essentially removes the first argument, the eventName, from TEventManager
type TEM<T extends TBaseEvents> = TEventManager<{ event: T }>
type TSingleEventManager<T extends TBaseEvents> = {
  [K in keyof TEM<T>]: TEM<T>[K] extends Function
    ? (arg: Parameters<TEM<T>[K]>[1]) => ReturnType<TEM<T>[K]>
    : never
}

export function createSingleEventManager<T extends TBaseEvents>(): TSingleEventManager<T> {
  const manager = createEventManager<{ event: T }>()

  return {
    subscribe(handler) {
      return manager.subscribe("event", handler)
    },
    isSubscribed(handler) {
      return manager.isSubscribed("event", handler)
    },
    unsubscribe(handler) {
      return manager.unsubscribe("event", handler)
    },
    publish(event) {
      return manager.publish("event", event)
    },
    clear() {
      return manager.clear("event")
    },
  }
}
