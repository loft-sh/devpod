import { exists, noop, Result, ResultError, Return, THandler } from "../../lib"
import {
  TUnsubscribeFn,
  TCacheID,
  TWorkspace,
  TWorkspaceID,
  TWorkspaceStartConfig,
  TWorkspaceWithoutStatus,
} from "../../types"
import { StartCommandCache } from "../cache"
import { TStreamEventListenerFn } from "../command"
import { TDebuggable } from "../types"
import { WorkspaceCommands } from "./workspaceCommands"

export class WorkspacesClient implements TDebuggable {
  constructor(private startCommandCache: StartCommandCache) {}

  public setDebug(isEnabled: boolean): void {
    WorkspaceCommands.DEBUG = isEnabled
  }

  private createStartHandler(
    id: TCacheID,
    listener: TStreamEventListenerFn | undefined
  ): THandler<TStreamEventListenerFn> {
    return {
      id,
      eq(other) {
        return id === other.id
      },
      notify: exists(listener) ? listener : noop,
    }
  }

  public async listAll(): Promise<Result<readonly TWorkspaceWithoutStatus[]>> {
    return WorkspaceCommands.ListWorkspaces()
  }

  public async getStatus(id: TWorkspaceID): Promise<Result<TWorkspace["status"]>> {
    const result = await WorkspaceCommands.GetWorkspaceStatus(id)
    if (result.err) {
      return result
    }

    const { status } = result.val

    return Return.Value(status)
  }

  public async newID(rawSource: string): Promise<Result<string>> {
    return await WorkspaceCommands.GetWorkspaceID(rawSource)
  }

  public async start(
    id: TWorkspaceID,
    config: TWorkspaceStartConfig,
    cacheID: TCacheID,
    listener?: TStreamEventListenerFn | undefined
  ): Promise<Result<TWorkspace["status"]>> {
    const maybeRunningCommand = this.startCommandCache.get(id)
    const handler = this.createStartHandler(cacheID, listener)

    // If `start` for id is running already,
    // wire up the new listener and return the existing operation
    if (exists(maybeRunningCommand)) {
      maybeRunningCommand.stream?.(handler)
      await maybeRunningCommand.promise

      return this.getStatus(id)
    }

    const cmd = WorkspaceCommands.StartWorkspace(id, config)
    const { operation, stream } = this.startCommandCache.connect(id, cmd)
    stream?.(handler)

    const result = await operation
    if (result.err) {
      return result
    }

    this.startCommandCache.clear(id)

    return this.getStatus(id)
  }

  public subscribeToStart(
    id: TWorkspaceID,
    viewID: TCacheID,
    listener?: TStreamEventListenerFn | undefined
  ): TUnsubscribeFn {
    const maybeRunningCommand = this.startCommandCache.get(id)
    if (!exists(maybeRunningCommand)) {
      return noop
    }

    const maybeUnsubscribe = maybeRunningCommand.stream?.(this.createStartHandler(viewID, listener))

    return () => maybeUnsubscribe?.()
  }

  public async stop(id: TWorkspaceID): Promise<Result<TWorkspace["status"]>> {
    const result = await WorkspaceCommands.StopWorkspace(id).run()
    if (result.err) {
      return result
    }

    return this.getStatus(id)
  }

  public async rebuild(id: TWorkspaceID): Promise<Result<TWorkspace["status"]>> {
    const result = await WorkspaceCommands.RebuildWorkspace(id).run()
    if (result.err) {
      return result
    }

    return this.getStatus(id)
  }

  public async remove(id: TWorkspaceID): Promise<ResultError> {
    return WorkspaceCommands.RemoveWorkspace(id).run()
  }
}
