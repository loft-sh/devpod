import { debug, EventManager, SingleEventManager } from "../../../lib"
import {
  TProID,
  TUnsubscribeFn,
  TWorkspace,
  TWorkspaceID,
  TWorkspaceWithoutStatus,
} from "../../../types"
import { ProWorkspaceInstance } from "../Pro"
import { Action, TActionFn, TActionName, TActionObj } from "../action"
import { ActionHistory } from "../action/actionHistory" // This is a workaround for how typescript resolves circular dependencies, usually the import should be from "./action"
import { replaceEqualDeep } from "../helpers"

type TLastActions = Readonly<{ active: readonly TActionObj[]; history: readonly TActionObj[] }>
type TStartActionArgs = Readonly<{
  actionName: TActionName
  workspaceKey: TInstanceID
  actionFn: TActionFn
}>

export interface IWorkspaceStore<TKey extends string, TW> {
  // workspaces
  get(id: TKey): TW | undefined
  getAll(): readonly TW[]
  setWorkspace(id: TKey, newWorkspace: TW): void
  setWorkspaces(newWorkspaces: readonly TW[]): void
  removeWorkspace(workspaceKey: TKey): void
  subscribe(listener: VoidFunction): TUnsubscribeFn
  setStatus(workspaceKey: TKey, status: string | null | undefined): void

  // workspace actions
  getAllActions(): TLastActions
  getCurrentAction(workspaceKey: TKey): TActionObj | undefined
  getWorkspaceActions(workspaceKey: TKey): TActionObj[]
  startAction(args: TStartActionArgs): Action["id"]
}

class InternalWorkspaceStore<TKey extends string, TWorkspace> {
  private readonly eventManager = new SingleEventManager<void>()
  private actionsHistory: ActionHistory
  private workspaces = new Map<TKey, TWorkspace>()
  private lastWorkspaces: readonly TWorkspace[] = []
  private lastActions: TLastActions = { active: [], history: [] }

  constructor(key?: string) {
    this.actionsHistory = new ActionHistory(key)
    this.lastActions = this.actionsHistory.getAll()
  }

  public subscribe(listener: VoidFunction): TUnsubscribeFn {
    const handler = EventManager.toHandler(listener)

    return this.eventManager.subscribe(handler)
  }

  public get(key: TKey): TWorkspace | undefined {
    return this.workspaces.get(key)
  }

  public getAll(): readonly TWorkspace[] {
    return this.lastWorkspaces
  }

  public getWorkspaceActions(workspaceKey: TKey): TActionObj[] {
    return [
      ...this.lastActions.active.filter((action) => action.targetID === workspaceKey),
      ...this.lastActions.history.filter((action) => action.targetID === workspaceKey).reverse(),
    ]
  }

  public getCurrentAction(workspaceKey: TKey): TActionObj | undefined {
    return this.lastActions.active.find((action) => action.targetID === workspaceKey)
  }

  public getAllActions(): TLastActions {
    return this.lastActions
  }

  public removeWorkspace(workspaceKey: TKey): void {
    this.workspaces.delete(workspaceKey)
    this.workspacesDidChange()
  }

  public startAction({ actionName, workspaceKey, actionFn }: TStartActionArgs): Action["id"] {
    // By default, actions cancel previous actios.
    // If you need to wait for an action to finish, you can use `getCurrentAction` and wait until it is undefined
    const maybeCurrentAction = this.actionsHistory.getActive(workspaceKey)
    if (maybeCurrentAction !== undefined) {
      maybeCurrentAction.cancel()
      this.actionsHistory.archive(maybeCurrentAction)
    }

    const action = new Action(actionName, workspaceKey, actionFn)
    this.actionsHistory.addActive(workspaceKey, action)

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

  public setWorkspaces(newWorkspaces: Map<TKey, TWorkspace>) {
    this.workspaces = newWorkspaces
    this.workspacesDidChange()
  }

  public setWorkspace(id: TKey, newWorkspace: TWorkspace) {
    this.workspaces.set(id, newWorkspace)
    this.workspacesDidChange()
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
}

export class WorkspaceStore implements IWorkspaceStore<TWorkspaceID, TWorkspace> {
  private store = new InternalWorkspaceStore<TWorkspaceID, TWorkspace>()

  public get(id: TWorkspaceID): TWorkspace | undefined {
    return this.store.get(id)
  }

  public getAll(): readonly TWorkspace[] {
    return this.store.getAll()
  }

  public setWorkspace(id: TWorkspaceID, newWorkspace: TWorkspace): void {
    return this.store.setWorkspace(id, newWorkspace)
  }

  public setWorkspaces(newWorkspaces: readonly TWorkspaceWithoutStatus[]): void {
    const prevWorkspaces = this.store.getAll().map((workspace) => {
      // we need to remove `status` before comparing the workspaces because the new ones will not have it.
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const { status: _, ...w } = workspace

      return w
    })

    const workspaces = replaceEqualDeep(prevWorkspaces, newWorkspaces)

    if (Object.is(workspaces, prevWorkspaces)) {
      return
    }

    const newWorkspacesMap = new Map(
      workspaces.map((workspace) => {
        // patch existing status if we have one for this workspace - new ones will be sent without it
        const maybeExistingWorkspace = this.store.get(workspace.id)

        return [workspace.id, { ...workspace, status: maybeExistingWorkspace?.status }]
      })
    )
    this.store.setWorkspaces(newWorkspacesMap)
  }

  public removeWorkspace(workspaceID: TWorkspaceID): void {
    return this.store.removeWorkspace(workspaceID)
  }

  public subscribe(listener: VoidFunction): TUnsubscribeFn {
    return this.store.subscribe(listener)
  }

  public setStatus(workspaceID: TWorkspaceID, status: string | null | undefined): void {
    const maybeWorkspace = this.store.get(workspaceID)
    if (maybeWorkspace === undefined) {
      return
    }

    const prevStatus = maybeWorkspace.status
    if (status === prevStatus) {
      return
    }

    this.store.setWorkspace(workspaceID, {
      ...maybeWorkspace,
      status: status as TWorkspace["status"],
    })
  }

  public getAllActions(): TLastActions {
    return this.store.getAllActions()
  }

  public getCurrentAction(workspaceID: TWorkspaceID): TActionObj | undefined {
    return this.store.getCurrentAction(workspaceID)
  }

  public getWorkspaceActions(workspaceID: TWorkspaceID): TActionObj[] {
    return this.store.getWorkspaceActions(workspaceID)
  }

  public startAction(args: TStartActionArgs): Action["id"] {
    return this.store.startAction(args)
  }
}

type TInstanceID = string
export class ProWorkspaceStore implements IWorkspaceStore<TInstanceID, ProWorkspaceInstance> {
  private store: InternalWorkspaceStore<TInstanceID, ProWorkspaceInstance>
  constructor(id: TProID) {
    this.store = new InternalWorkspaceStore<TInstanceID, ProWorkspaceInstance>(id)
  }

  public get(key: TInstanceID): ProWorkspaceInstance | undefined {
    return this.store.get(key)
  }

  public getAll(): readonly ProWorkspaceInstance[] {
    return this.store.getAll()
  }

  public setWorkspace(key: TInstanceID, newWorkspace: ProWorkspaceInstance): void {
    return this.store.setWorkspace(key, newWorkspace)
  }

  public setWorkspaces(newInstances: readonly ProWorkspaceInstance[]): void {
    const prevInstances = this.store.getAll()

    const instances = replaceEqualDeep(prevInstances, newInstances)

    if (Object.is(instances, prevInstances)) {
      return
    }

    const newWorkspacesMap = new Map(instances.map((instance) => [instance.id, instance]))
    this.store.setWorkspaces(newWorkspacesMap)
  }

  public removeWorkspace(workspaceKey: TInstanceID): void {
    return this.store.removeWorkspace(workspaceKey)
  }

  public subscribe(listener: VoidFunction): TUnsubscribeFn {
    return this.store.subscribe(listener)
  }

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  public setStatus(_workspaceKey: TInstanceID, _status: string): void {
    // noop
    return
  }

  public getAllActions(): TLastActions {
    return this.store.getAllActions()
  }

  public getCurrentAction(workspaceKey: TInstanceID): TActionObj | undefined {
    return this.store.getCurrentAction(workspaceKey)
  }

  public getWorkspaceActions(workspaceKey: TInstanceID): TActionObj[] {
    return this.store.getWorkspaceActions(workspaceKey)
  }

  public startAction(args: TStartActionArgs): Action["id"] {
    return this.store.startAction(args)
  }
}
