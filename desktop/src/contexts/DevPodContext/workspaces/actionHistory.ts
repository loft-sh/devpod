import { TWorkspaceID } from "../../../types"
import { Action, TActionObj, TWorkspaceActions } from "./action"

const HISTORY_KEY = "devpod-workspace-action-history"
const MAX_HISTORY_ENTRIES = 50

export class ActionHistory {
  private active = new Map<TWorkspaceID, Action>()
  private history: TActionObj[]

  constructor() {
    const maybeHistory = localStorage.getItem(HISTORY_KEY)
    if (maybeHistory === null) {
      this.history = []

      return
    }

    this.history = JSON.parse(maybeHistory) as TActionObj[]
  }

  private getAllActive(): readonly Action[] {
    return Array.from(this.active.values())
  }

  public getActive(workspaceID: TWorkspaceID): Action | undefined {
    return this.active.get(workspaceID)
  }

  public getAll(): TWorkspaceActions {
    return { active: this.getAllActive(), history: [...this.history] }
  }

  public addActive(workspaceID: TWorkspaceID, action: Action): void {
    this.active.set(workspaceID, action)
  }

  public archive(action: Action): void {
    this.active.delete(action.workspaceID)
    this.history.push(action.getData())

    // Limit history size
    const overflow = this.history.length - MAX_HISTORY_ENTRIES
    if (overflow > 0) {
      this.history.splice(0, overflow)
    }

    window.localStorage.setItem(HISTORY_KEY, JSON.stringify(this.history))
  }
}
