import { invoke } from "@tauri-apps/api"
import { TActionID, TActionName, TPublicAction } from "../../contexts"
import { exists, noop, Result, ResultError, Return, THandler } from "../../lib"
import {
  TStreamID,
  TUnsubscribeFn,
  TWorkspace,
  TWorkspaceID,
  TWorkspaceStartConfig,
  TWorkspaceWithoutStatus,
} from "../../types"
import { TCommand, TStreamEventListenerFn } from "../command"
import { CommandCache, TCommandCacheInfo } from "../commandCache"
import { TDebuggable, TStreamEvent } from "../types"
import { WorkspaceCommands } from "./workspaceCommands"

// Every workspace can have one active action at a time,
// but multiple views might need to listen to the same action.
// The `streamID` identifies a view listener.
type TWorkspaceClientContext = Readonly<{
  id: TWorkspaceID
  actionID: TActionID
  streamID: TStreamID
}>

export class WorkspacesClient implements TDebuggable {
  private readonly commandCache = new CommandCache()

  constructor() {}

  private createStreamHandler(
    id: TStreamID,
    listener: TStreamEventListenerFn
  ): THandler<TStreamEventListenerFn> {
    return {
      id,
      eq(other) {
        return id === other.id
      },
      notify: listener,
    }
  }

  private async writeEvent(actionID: TActionID, event: TStreamEvent) {
    // Be wary of the spelling, tauri expects this to be `actionId` instead of `actionID` because of the serde deserialization
    await invoke("write_action_log", { actionId: actionID, data: JSON.stringify(event) })
  }

  private async execActionCmd<T>(
    cmd: Readonly<TCommand<T>>,
    ctx: Readonly<{
      id: TWorkspaceID
      actionID: TActionID
      streamID: TStreamID
      listener?: TStreamEventListenerFn | undefined
      actionName: TActionName
    }>
  ) {
    const cacheInfo: TCommandCacheInfo = { id: ctx.id, actionName: ctx.actionName }
    const maybeRunningCommand = this.commandCache.get(cacheInfo)
    const handler = this.createStreamHandler(ctx.streamID, (event) => {
      this.writeEvent(ctx.actionID, event)

      ctx.listener?.(event)
    })

    // If `start` for id is running already,
    // wire up the new listener and return the existing operation
    if (exists(maybeRunningCommand)) {
      maybeRunningCommand.stream?.(handler)
      await maybeRunningCommand.promise

      return this.getStatus(ctx.id)
    }

    const { operation, stream } = this.commandCache.connect(cacheInfo, cmd)
    stream?.(handler)

    const result = await operation
    if (result.err) {
      return result
    }

    this.commandCache.clear(cacheInfo)

    return this.getStatus(ctx.id)
  }

  public setDebug(isEnabled: boolean): void {
    WorkspaceCommands.DEBUG = isEnabled
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
    config: TWorkspaceStartConfig,
    listener: TStreamEventListenerFn | undefined,
    ctx: TWorkspaceClientContext
  ): Promise<Result<TWorkspace["status"]>> {
    const cmd = WorkspaceCommands.StartWorkspace(ctx.id, config)

    return this.execActionCmd(cmd, { ...ctx, listener, actionName: "start" })
  }

  public async stop(
    listener: TStreamEventListenerFn | undefined,
    ctx: TWorkspaceClientContext
  ): Promise<Result<TWorkspace["status"]>> {
    const cmd = WorkspaceCommands.StopWorkspace(ctx.id)

    return this.execActionCmd(cmd, { ...ctx, listener, actionName: "stop" })
  }

  public async rebuild(
    listener: TStreamEventListenerFn | undefined,
    ctx: TWorkspaceClientContext
  ): Promise<Result<TWorkspace["status"]>> {
    const cmd = WorkspaceCommands.RebuildWorkspace(ctx.id)

    return this.execActionCmd(cmd, { ...ctx, listener, actionName: "rebuild" })
  }

  public async remove(
    listener: TStreamEventListenerFn | undefined,
    ctx: TWorkspaceClientContext
  ): Promise<ResultError> {
    const cmd = WorkspaceCommands.RemoveWorkspace(ctx.id)

    const result = await this.execActionCmd(cmd, { ...ctx, listener, actionName: "remove" })

    if (result.err) {
      return result
    }

    return Return.Ok()
  }

  // TODO: Make nicer :)
  public removeMany(workspaces: readonly TWorkspace[]) {
    for (const workspace of workspaces) {
      WorkspaceCommands.RemoveWorkspace(workspace.id).run()
    }
  }

  public subscribe(
    action: TPublicAction,
    streamID: TStreamID,
    listener: TStreamEventListenerFn
  ): TUnsubscribeFn {
    const maybeRunningCommand = this.commandCache.get({
      id: action.workspaceID,
      actionName: action.name,
    })
    if (!exists(maybeRunningCommand)) {
      return noop
    }

    const maybeUnsubscribe = maybeRunningCommand.stream?.(
      this.createStreamHandler(streamID, listener)
    )

    return () => maybeUnsubscribe?.()
  }

  public replayAction(actionID: TActionID, listener: TStreamEventListenerFn): TUnsubscribeFn {
    let cancelled = false
    const unsubscribe = () => {
      cancelled = true
    }
    // Be wary of the spelling, tauri expects this to be `actionId` instead of `actionID` because of the serde deserialization
    invoke<readonly string[]>("get_action_logs", { actionId: actionID })
      .then((events) => {
        if (cancelled) {
          return
        }
        for (const event of events) {
          try {
            listener(JSON.parse(event))
          } catch (e) {
            console.log(e)
            // noop
          }
        }
      })
      .catch((e) => {
        console.error("Failed to replay action", e)
        unsubscribe()
      })

    return unsubscribe
  }
}
