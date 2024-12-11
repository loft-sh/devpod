import { Action, TActionObj, TActions } from "./action"

const HISTORY_KEY = "devpod-workspace-action-history"
const MAX_HISTORY_ENTRIES = 50

export class ActionHistory {
  private active = new Map<string, Action>()
  private history: TActionObj[]

  private localStorageKey: string

  constructor(keySuffix?: string) {
    let localStorageKey = HISTORY_KEY
    if (keySuffix) {
      localStorageKey = `${localStorageKey}-${keySuffix}`
    }
    this.localStorageKey = localStorageKey

    const maybeHistory = localStorage.getItem(this.localStorageKey)
    if (maybeHistory === null) {
      this.history = []

      return
    }

    this.history = JSON.parse(maybeHistory) as TActionObj[]
  }

  private getAllActive(): readonly TActionObj[] {
    const active = []
    for (const action of this.active.values()) {
      active.push(action.getData())
    }

    return active
  }

  public getActive(targetID: string): Action | undefined {
    return this.active.get(targetID)
  }

  public getAll(): TActions {
    const active = this.getAllActive()
    const history = this.history.slice()

    return { active, history }
  }

  public addActive(targetID: string, action: Action): void {
    this.active.set(targetID, action)
  }

  public archive(action: Action): void {
    this.active.delete(action.targetID)
    this.history.push(action.getData())

    // Limit history size
    const overflow = this.history.length - MAX_HISTORY_ENTRIES
    if (overflow > 0) {
      this.history.splice(0, overflow)
    }

    window.localStorage.setItem(this.localStorageKey, JSON.stringify(this.history))
  }
}
