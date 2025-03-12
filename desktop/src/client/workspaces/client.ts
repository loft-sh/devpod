import { TActionID, TActionName, TActionObj } from "../../contexts"
import { Result, ResultError, Return, THandler, exists, isError, noop } from "../../lib"
import {
  TDevcontainerSetup,
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
import {
  DEVPOD_FLAG_DOTFILES,
  DEVPOD_FLAG_GIT_SIGNING_KEY,
  WORKSPACE_COMMAND_ADDITIONAL_FLAGS_KEY,
} from "../constants"
import { invoke } from "@tauri-apps/api/core"

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
    this.commandCache.clear(cacheInfo)

    if (result.err) {
      return result
    }

    return result
  }

  public setDebug(isEnabled: boolean): void {
    WorkspaceCommands.DEBUG = isEnabled
  }

  public setDotfilesFlag(dotfilesUrl: string): void {
    if (!dotfilesUrl) {
      return
    }
    WorkspaceCommands.ADDITIONAL_FLAGS.set(DEVPOD_FLAG_DOTFILES, dotfilesUrl)
  }

  public setAdditionalFlags(additionalFlags: string): void {
    WorkspaceCommands.ADDITIONAL_FLAGS.set(WORKSPACE_COMMAND_ADDITIONAL_FLAGS_KEY, additionalFlags)
  }

  public setSSHKeyPath(sshKeyPath: string): void {
    if (!sshKeyPath) {
      return
    }
    WorkspaceCommands.ADDITIONAL_FLAGS.set(DEVPOD_FLAG_GIT_SIGNING_KEY, sshKeyPath)
  }

  public async listAll(
    skipPro: boolean = true
  ): Promise<Result<readonly TWorkspaceWithoutStatus[]>> {
    return WorkspaceCommands.ListWorkspaces(skipPro)
  }

  public async getStatus(id: TWorkspaceID): Promise<Result<TWorkspace["status"]>> {
    const result = await WorkspaceCommands.FetchWorkspaceStatus(id)
    if (result.err) {
      return result
    }

    const { status } = result.val

    return Return.Value(status)
  }

  public async newID(rawSource: string): Promise<Result<string>> {
    return WorkspaceCommands.GetWorkspaceID(rawSource)
  }

  public async newUID(): Promise<Result<string>> {
    return WorkspaceCommands.GetWorkspaceUID()
  }

  public async start(
    config: TWorkspaceStartConfig,
    listener: TStreamEventListenerFn | undefined,
    ctx: TWorkspaceClientContext
  ): Promise<Result<TWorkspace["status"]>> {
    const cmd = WorkspaceCommands.StartWorkspace(ctx.id, config)
    const result = await this.execActionCmd(cmd, { ...ctx, listener, actionName: "start" })
    if (result.err) {
      return result
    }

    return this.getStatus(ctx.id)
  }

  public async stop(
    listener: TStreamEventListenerFn | undefined,
    ctx: TWorkspaceClientContext
  ): Promise<Result<TWorkspace["status"]>> {
    const cmd = WorkspaceCommands.StopWorkspace(ctx.id)
    const result = await this.execActionCmd(cmd, { ...ctx, listener, actionName: "stop" })
    if (result.err) {
      return result
    }

    return this.getStatus(ctx.id)
  }

  public async rebuild(
    listener: TStreamEventListenerFn | undefined,
    ctx: TWorkspaceClientContext
  ): Promise<Result<TWorkspace["status"]>> {
    const cmd = WorkspaceCommands.RebuildWorkspace(ctx.id)
    const result = await this.execActionCmd(cmd, { ...ctx, listener, actionName: "rebuild" })
    if (result.err) {
      return result
    }

    return this.getStatus(ctx.id)
  }

  public async troubleshoot(ctx: TWorkspaceClientContext) {
    const cmd = WorkspaceCommands.TroubleshootWorkspace(ctx.id)

    return cmd.run()
  }

  public async reset(
    listener: TStreamEventListenerFn | undefined,
    ctx: TWorkspaceClientContext
  ): Promise<Result<TWorkspace["status"]>> {
    const cmd = WorkspaceCommands.ResetWorkspace(ctx.id)
    const result = await this.execActionCmd(cmd, { ...ctx, listener, actionName: "reset" })
    if (result.err) {
      return result
    }

    return this.getStatus(ctx.id)
  }

  public async remove(
    force: boolean,
    listener: TStreamEventListenerFn | undefined,
    ctx: TWorkspaceClientContext
  ): Promise<Result<TWorkspace["status"]>> {
    const cmd = WorkspaceCommands.RemoveWorkspace(ctx.id, force)
    const result = await this.execActionCmd(cmd, { ...ctx, listener, actionName: "remove" })
    if (result.err) {
      return result
    }

    return result
  }

  public async checkStatus(
    listener: TStreamEventListenerFn | undefined,
    ctx: TWorkspaceClientContext
  ): Promise<ResultError> {
    const cmd = WorkspaceCommands.GetStatusLogs(ctx.id)
    const result = await this.execActionCmd(cmd, { ...ctx, listener, actionName: "checkStatus" })
    if (result.err) {
      return result
    }

    return Return.Ok()
  }

  public async checkDevcontainerSetup(rawSource: string): Promise<Result<TDevcontainerSetup>> {
    const result = await WorkspaceCommands.GetDevcontainerConfig(rawSource).run()
    if (result.err) {
      return result
    }

    try {
      const setup = JSON.parse(result.val.stdout) as TDevcontainerSetup

      return Return.Value(setup)
    } catch (err) {
      return Return.Failed(`Failed to parse devcontainer setup: ${err}`)
    }
  }

  public subscribe(
    action: TActionObj,
    streamID: TStreamID,
    listener: TStreamEventListenerFn
  ): TUnsubscribeFn {
    const maybeRunningCommand = this.commandCache.get({
      id: action.targetID,
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

  public async cancelAction(actionID: TActionID): Promise<ResultError> {
    const cmdHandler = this.commandCache.findCommandHandlerById(actionID)

    return cmdHandler?.cancel?.() ?? Return.Ok()
  }

  public async getActionLogFile(actionID: TActionID): Promise<Result<string>> {
    try {
      const path = await invoke<string>("get_action_log_file", { actionId: actionID })

      return Return.Value(path)
    } catch (e) {
      if (isError(e)) {
        return Return.Failed(e.message)
      }

      return Return.Failed(`Unable to retrieve log file for action ${actionID}`)
    }
  }
}
