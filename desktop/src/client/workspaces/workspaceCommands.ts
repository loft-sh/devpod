import { exists, Result, Return } from "../../lib"
import {
  TWorkspace,
  TWorkspaceID,
  TWorkspaceStartConfig,
  TWorkspaceStatusResult,
  TWorkspaceWithoutStatus,
} from "../../types"
import { Command, isOk, toFlagArg, toMultipleFlagArg } from "../command"
import {
  DEVPOD_COMMAND_DELETE,
  DEVPOD_COMMAND_GET_WORKSPACE_CONFIG,
  DEVPOD_COMMAND_GET_WORKSPACE_NAME,
  DEVPOD_COMMAND_HELPER,
  DEVPOD_COMMAND_LIST,
  DEVPOD_COMMAND_STATUS,
  DEVPOD_COMMAND_STOP,
  DEVPOD_COMMAND_UP,
  DEVPOD_FLAG_DEBUG,
  DEVPOD_FLAG_DEVCONTAINER_PATH,
  DEVPOD_FLAG_FORCE,
  DEVPOD_FLAG_GIT_BRANCH,
  DEVPOD_FLAG_GIT_COMMIT,
  DEVPOD_FLAG_ID,
  DEVPOD_FLAG_IDE,
  DEVPOD_FLAG_JSON_LOG_OUTPUT,
  DEVPOD_FLAG_JSON_OUTPUT,
  DEVPOD_FLAG_PREBUILD_REPOSITORY,
  DEVPOD_FLAG_PROVIDER,
  DEVPOD_FLAG_RECREATE,
  DEVPOD_FLAG_RESET,
  DEVPOD_FLAG_TIMEOUT,
} from "../constants"

type TRawWorkspaces = readonly (Omit<TWorkspace, "status" | "id"> &
  Readonly<{ id: string | null }>)[]

export class WorkspaceCommands {
  static DEBUG = false
  static ADDITIONAL_FLAGS = ""

  private static newCommand(args: string[]): Command {
    return new Command([...args, ...(WorkspaceCommands.DEBUG ? [DEVPOD_FLAG_DEBUG] : [])])
  }

  static async ListWorkspaces(): Promise<Result<TWorkspaceWithoutStatus[]>> {
    const result = await new Command([DEVPOD_COMMAND_LIST, DEVPOD_FLAG_JSON_OUTPUT]).run()
    if (result.err) {
      return result
    }

    const rawWorkspaces = JSON.parse(result.val.stdout) as TRawWorkspaces

    return Return.Value(
      rawWorkspaces.filter((workspace): workspace is TWorkspaceWithoutStatus =>
        exists(workspace.id)
      )
    )
  }

  static async FetchWorkspaceStatus(
    id: string
  ): Promise<Result<Pick<TWorkspace, "id" | "status">>> {
    const result = await new Command([DEVPOD_COMMAND_STATUS, id, DEVPOD_FLAG_JSON_OUTPUT]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return Return.Failed(`Failed to get status for workspace ${id}: ${result.val.stderr}`)
    }

    const { state } = JSON.parse(result.val.stdout) as TWorkspaceStatusResult

    return Return.Value({ id, status: state })
  }

  static async GetWorkspaceID(source: string) {
    const result = await new Command([
      DEVPOD_COMMAND_HELPER,
      DEVPOD_COMMAND_GET_WORKSPACE_NAME,
      source,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return Return.Failed(`Failed to get ID for workspace source ${source}: ${result.val.stderr}`)
    }

    return Return.Value(result.val.stdout)
  }

  static GetStatusLogs(id: string) {
    return new Command([DEVPOD_COMMAND_STATUS, id, DEVPOD_FLAG_JSON_LOG_OUTPUT])
  }

  static StartWorkspace(id: TWorkspaceID, config: TWorkspaceStartConfig) {
    const maybeSource = config.sourceConfig?.source
    const maybeIDFlag = exists(maybeSource) ? [toFlagArg(DEVPOD_FLAG_ID, id)] : []

    const maybeIdeName = config.ideConfig?.name
    const maybeIDEFlag = exists(maybeIdeName) ? [toFlagArg(DEVPOD_FLAG_IDE, maybeIdeName)] : []

    const maybeProviderID = config.providerConfig?.providerID
    const maybeProviderFlag = exists(maybeProviderID)
      ? [toFlagArg(DEVPOD_FLAG_PROVIDER, maybeProviderID)]
      : []

    const maybePrebuildRepositories = config.prebuildRepositories?.length
      ? [toFlagArg(DEVPOD_FLAG_PREBUILD_REPOSITORY, config.prebuildRepositories.join(","))]
      : []

    const maybeDevcontainerPath = config.devcontainerPath
      ? [toFlagArg(DEVPOD_FLAG_DEVCONTAINER_PATH, config.devcontainerPath)]
      : []

    const identifier = exists(maybeSource) && exists(maybeIDFlag) ? maybeSource : id

    const additionalFlags =
      WorkspaceCommands.ADDITIONAL_FLAGS.length !== 0
        ? toMultipleFlagArg(WorkspaceCommands.ADDITIONAL_FLAGS)
        : []

    const maybeGitBranch = config.sourceConfig?.gitBranch
    const gitBranchFlag = exists(maybeGitBranch)
      ? [toFlagArg(DEVPOD_FLAG_GIT_BRANCH, maybeGitBranch)]
      : []

    const maybeGitCommit = config.sourceConfig?.gitCommit
    const gitCommitFlag = exists(maybeGitCommit)
      ? [toFlagArg(DEVPOD_FLAG_GIT_COMMIT, maybeGitCommit)]
      : []

    return WorkspaceCommands.newCommand([
      DEVPOD_COMMAND_UP,
      identifier,
      ...maybeIDFlag,
      ...maybeIDEFlag,
      ...maybeProviderFlag,
      ...maybePrebuildRepositories,
      ...maybeDevcontainerPath,
      ...gitBranchFlag,
      ...gitCommitFlag,
      ...additionalFlags,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
    ])
  }

  static StopWorkspace(id: TWorkspaceID) {
    return WorkspaceCommands.newCommand([DEVPOD_COMMAND_STOP, id, DEVPOD_FLAG_JSON_LOG_OUTPUT])
  }

  static RebuildWorkspace(id: TWorkspaceID) {
    return WorkspaceCommands.newCommand([
      DEVPOD_COMMAND_UP,
      id,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
      DEVPOD_FLAG_RECREATE,
    ])
  }

  static ResetWorkspace(id: TWorkspaceID) {
    return WorkspaceCommands.newCommand([
      DEVPOD_COMMAND_UP,
      id,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
      DEVPOD_FLAG_RESET,
    ])
  }

  static RemoveWorkspace(id: TWorkspaceID, force?: boolean) {
    const args = [DEVPOD_COMMAND_DELETE, id, DEVPOD_FLAG_JSON_LOG_OUTPUT]
    if (force) {
      args.push(DEVPOD_FLAG_FORCE)
    }

    return WorkspaceCommands.newCommand(args)
  }

  static GetDevcontainerConfig(rawSource: string) {
    return new Command([
      DEVPOD_COMMAND_HELPER,
      DEVPOD_COMMAND_GET_WORKSPACE_CONFIG,
      rawSource,
      DEVPOD_FLAG_TIMEOUT,
      "10s",
    ])
  }
}
