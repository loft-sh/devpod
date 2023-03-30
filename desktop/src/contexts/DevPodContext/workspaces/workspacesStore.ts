import { replaceEqualDeep } from "@tanstack/react-query"
import { debug, EventManager, SingleEventManager } from "../../../lib"
import { TUnsubscribeFn, TWorkspace, TWorkspaceID } from "../../../types"
import { Action, TActionFn, TActionName } from "./action"

// We don't want to expose the methods to consumers of these actions, so we'll strip them off on the type level
type TPublicAction = Omit<Action, "run" | "cancel" | "once">

class WorkspacesStore {
  private readonly eventManager = new SingleEventManager<void>()
  private history: Action[] = []
  private workspaces = new Map<TWorkspaceID, TWorkspace>()
  private actions = new Map<TWorkspaceID, Action>()
  private lastWorkspaces: readonly TWorkspace[] = []
  private lastActions: readonly TPublicAction[] = []

  constructor() {}

  private actionDidChange() {
    this.lastActions = Array.from(this.actions.values())
    debug("actions", this.lastActions)
    this.eventManager.publish()
  }

  private workspacesDidChange() {
    this.lastWorkspaces = Array.from(this.workspaces.values())
    debug("workspaces", this.lastWorkspaces)
    this.eventManager.publish()
  }

  public subscribe(listener: VoidFunction): TUnsubscribeFn {
    const handler = EventManager.toHandler(listener)

    return this.eventManager.subscribe(handler)
  }

  public get(id: TWorkspaceID): TWorkspace | undefined {
    return this.workspaces.get(id)
  }

  public getAll(): readonly TWorkspace[] {
    return this.lastWorkspaces
  }

  public getCurrentAction(workspaceID: TWorkspaceID): TPublicAction | undefined {
    return this.actions.get(workspaceID)
  }

  public getAllActions(): readonly TPublicAction[] {
    return this.lastActions
  }

  public setWorkspaces(newWorkspaces: readonly TWorkspace[]) {
    const prevWorkspaces = this.lastWorkspaces
    const workspaces = replaceEqualDeep(prevWorkspaces, newWorkspaces)

    if (Object.is(workspaces, prevWorkspaces)) {
      return
    }

    this.workspaces = new Map(
      workspaces.map((workspace) => {
        // patch existing status if we have one for this workspace - new ones will be sent without it
        const maybeExistingWorkspace = this.workspaces.get(workspace.id)

        return [workspace.id, { ...workspace, status: maybeExistingWorkspace?.status }]
      })
    )
    this.workspacesDidChange()
  }

  public setStatus(workspaceID: TWorkspaceID, status: TWorkspace["status"]) {
    const maybeWorkspace = this.workspaces.get(workspaceID)
    if (maybeWorkspace === undefined) {
      return
    }

    const prevStatus = maybeWorkspace.status
    if (status === prevStatus) {
      return
    }

    this.workspaces.set(workspaceID, { ...maybeWorkspace, status })
    this.eventManager.publish()
  }

  public startAction({
    actionName,
    workspaceID,
    actionFn,
  }: Readonly<{
    actionName: TActionName
    workspaceID: TWorkspaceID
    actionFn: TActionFn
  }>): void {
    // By default, actions cancel previous actios.
    // If you need to wait for an action to finish, you can use `getCurrentAction` and wait until it is undefined
    const maybeCurrentAction = this.actions.get(workspaceID)
    if (maybeCurrentAction !== undefined) {
      maybeCurrentAction.cancel()
      this.history.push(maybeCurrentAction)
      this.actions.delete(maybeCurrentAction.workpaceID)
    }

    const action = new Action(actionName, workspaceID, actionFn)
    this.actions.set(workspaceID, action)

    // Setup listener for when the action is done
    action.once(() => {
      this.history.push(action)
      this.actions.delete(action.workpaceID)
      this.actionDidChange()
    })

    action.run()
    this.actionDidChange()
  }
}

// Singleton store
export const workspacesStore = new WorkspacesStore()
