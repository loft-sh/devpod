import { Result, SingleEventManager, EventManager } from "../../../lib"
import { TWorkspaceID } from "../../../types"
import { TWorkspaceResult } from "./useWorkspace"

export type TActionName = keyof Pick<
  TWorkspaceResult,
  "create" | "start" | "stop" | "rebuild" | "remove"
>
export type TActionFn = (context: TActionContext) => Promise<Result<unknown>>
export type TActionStatus = "pending" | "success" | "error" | "cancelled"
export type TAction = typeof Action
// We don't want to expose the methods to consumers of these actions, so we'll strip them off on the type level
export type TPublicAction = Omit<Action, "run" | "cancel" | "once">
type TActionContext = Readonly<{ id: Action["id"] }>

export class Action {
  private _status: TActionStatus = "pending"
  private _error: Error | undefined = undefined
  private createdAt = Date.now()
  private readonly eventManager = new SingleEventManager<TActionStatus>()
  public readonly id = window.crypto.randomUUID()

  constructor(
    public readonly name: TActionName,
    public readonly workpaceID: TWorkspaceID,
    private actionFn: TActionFn
  ) {}

  public get status() {
    return this._status
  }

  public get error() {
    return this._error
  }

  private failed(error: Error) {
    this._status = "error"
    this._error = error
    this.eventManager.publish(this._status)
  }

  private succeeded() {
    this._status = "success"
    this.eventManager.publish(this._status)
  }

  public run() {
    // TODO: Cancel somehow?

    this.actionFn({ id: this.id }).then((result) => {
      if (result.err) {
        this.failed(result.val)

        return
      }

      this.succeeded()
    })
  }

  public cancel() {
    // We're no longer interested in status updates
    this.eventManager.clear()
    // TODO: Implement
  }

  public once(listener: (status: TActionStatus) => void): void {
    const unsubscribe = this.eventManager.subscribe(
      EventManager.toHandler((status) => {
        listener(status)
        unsubscribe()
      })
    )
  }
}
