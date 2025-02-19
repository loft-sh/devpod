import { Result, SingleEventManager, EventManager } from "../../../lib"
import { v4 as uuidv4 } from "uuid"

export type TActionName = "start" | "stop" | "rebuild" | "reset" | "remove" | "checkStatus"
export type TActionFn = (context: TActionContext) => Promise<Result<unknown>>
export type TActionStatus = "pending" | "success" | "error" | "cancelled"
export type TActionID = Action["id"]
// We don't want to expose the methods to consumers of these actions, so we'll limit the type to data-only properties
export type TActionObj = Pick<
  Action,
  "id" | "name" | "status" | "error" | "createdAt" | "finishedAt" | "targetID"
>
export type TActions = Readonly<{
  active: readonly TActionObj[]
  history: readonly TActionObj[]
}>
type TActionContext = Readonly<{ id: Action["id"] }>

export class Action {
  private _status: TActionStatus = "pending"
  private _error: Error | undefined = undefined
  private _finishedAt: number | undefined = undefined
  private readonly eventManager = new SingleEventManager<TActionStatus>()
  public readonly id = uuidv4()
  public readonly createdAt = Date.now()

  public static deserialize(str: string): TActionObj {
    return JSON.parse(str)
  }

  constructor(
    public readonly name: TActionName,
    public readonly targetID: string,
    private actionFn: TActionFn
  ) {}

  public get status() {
    return this._status
  }

  public get error() {
    return this._error
  }

  public get finishedAt() {
    return this._finishedAt
  }

  private failed(error: Error) {
    if (this._status !== "pending") {
      return
    }
    this._status = "error"
    this._error = error
    this._finishedAt = Date.now()
    this.eventManager.publish(this._status)
  }

  private succeeded() {
    if (this._status !== "pending") {
      return
    }
    this._status = "success"
    this._finishedAt = Date.now()
    this.eventManager.publish(this._status)
  }

  public run() {
    this.actionFn({ id: this.id }).then((result) => {
      if (result.err) {
        this.failed(result.val)

        return
      }

      this.succeeded()
    })
  }

  public cancel() {
    if (this._status !== "pending") {
      return
    }
    // We're no longer interested in status updates
    this.eventManager.clear()
    this._status = "cancelled"
    this._finishedAt = Date.now()
  }

  public once(listener: (status: TActionStatus) => void): void {
    const unsubscribe = this.eventManager.subscribe(
      EventManager.toHandler((status) => {
        listener(status)
        unsubscribe()
      })
    )
  }

  public getData(): TActionObj {
    return {
      id: this.id,
      targetID: this.targetID,
      name: this.name,
      status: this.status,
      error: this.error,
      createdAt: this.createdAt,
      finishedAt: this.finishedAt,
    }
  }
}
