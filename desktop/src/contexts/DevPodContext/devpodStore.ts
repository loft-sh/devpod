import { debug, EventManager, SingleEventManager } from "../../lib"
import { TUnsubscribeFn, TWorkspace, TWorkspaceID, TWorkspaceWithoutStatus } from "../../types"
import { replaceEqualDeep } from "./helpers"
import { Action, TActionFn, TActionName, TActionObj } from "./action"
import { ActionHistory } from "./action/actionHistory" // This is a workaround for how typescript resolves circular dependencies, usually the import should be from "./action"

type TLastActions = Readonly<{ active: readonly TActionObj[]; history: readonly TActionObj[] }>

class DevpodStore {
  private readonly eventManager = new SingleEventManager<void>()
  private actionsHistory = new ActionHistory()
  private workspaces = new Map<TWorkspaceID, TWorkspace>()
  private lastWorkspaces: readonly TWorkspace[] = []
  private lastActions: TLastActions = { active: [], history: [] }

  constructor() {
    this.lastActions = this.actionsHistory.getAll()
  }

  private actionDidChange() {
    this.lastActions = this.actionsHistory.getAll()
    this.eventManager.publish()
    debug("actions", this.lastActions)
  }

  private workspacesDidChange() {
    this.lastWorkspaces = Array.from(this.workspaces.values())
    this.eventManager.publish()
    debug("workspaces", this.lastWorkspaces)
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

  public getCurrentAction(workspaceID: TWorkspaceID): TActionObj | undefined {
    return this.lastActions.active.find((action) => action.targetID === workspaceID)
  }

  public getAllActions(): TLastActions {
    return this.lastActions
  }

  public setWorkspaces(newWorkspaces: readonly TWorkspaceWithoutStatus[]): void {
    const prevWorkspaces = this.lastWorkspaces.map((workspace) => {
      // we need to remove `status` before comparing the workspaces because the new ones will not have it.
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const { status: _, ...w } = workspace

      return w
    })

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

  public removeWorkspace(workspaceID: TWorkspaceID): void {
    this.workspaces.delete(workspaceID)
    this.workspacesDidChange()
  }

  public setStatus(workspaceID: TWorkspaceID, status: TWorkspace["status"]): void {
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
  }>): Action["id"] {
    // By default, actions cancel previous actios.
    // If you need to wait for an action to finish, you can use `getCurrentAction` and wait until it is undefined
    const maybeCurrentAction = this.actionsHistory.getActive(workspaceID)
    if (maybeCurrentAction !== undefined) {
      maybeCurrentAction.cancel()
      this.actionsHistory.archive(maybeCurrentAction)
    }

    const action = new Action(actionName, workspaceID, actionFn)
    this.actionsHistory.addActive(workspaceID, action)

    // Setup listener for when the action is done
    action.once(() => {
      // We need to give the UI a chance to listen to the settled state, so we need to inform it about the change once
      // before and once after archiving the action
      this.actionDidChange()
      this.actionsHistory.archive(action)
      // Notify react on next tick of event loop to check the actions once more.
      // This ensures we have a chance to move from the `pending` to one of the `settled` states with the UI noticing.
      setTimeout(() => {
        this.actionDidChange()
      }, 0)
    })

    action.run()
    this.actionDidChange()

    return action.id
  }
}

// Singleton store
export const devpodStore = new DevpodStore()
