import { exists, Result, Return } from "../../lib"
import {
  TWorkspace,
  TWorkspaceID,
  TWorkspaceStartConfig,
  TWorkspaceStatusResult,
  TWorkspaceWithoutStatus,
} from "../../types"
import { Command, isOk, serializeRawOptions, toFlagArg } from "../command"
import {
  DEVPOD_COMMAND_DELETE,
  DEVPOD_COMMAND_GET_WORKSPACE_CONFIG,
  DEVPOD_COMMAND_GET_WORKSPACE_NAME,
  DEVPOD_COMMAND_GET_WORKSPACE_UID,
  DEVPOD_COMMAND_HELPER,
  DEVPOD_COMMAND_LIST,
  DEVPOD_COMMAND_STATUS,
  DEVPOD_COMMAND_STOP,
  DEVPOD_COMMAND_UP,
  DEVPOD_COMMAND_TROUBLESHOOT,
  DEVPOD_FLAG_DEBUG,
  DEVPOD_FLAG_DEVCONTAINER_PATH,
  DEVPOD_FLAG_FORCE,
  DEVPOD_FLAG_ID,
  DEVPOD_FLAG_IDE,
  DEVPOD_FLAG_JSON_LOG_OUTPUT,
  DEVPOD_FLAG_JSON_OUTPUT,
  DEVPOD_FLAG_PREBUILD_REPOSITORY,
  DEVPOD_FLAG_PROVIDER,
  DEVPOD_FLAG_PROVIDER_OPTION,
  DEVPOD_FLAG_RECREATE,
  DEVPOD_FLAG_RESET,
  DEVPOD_FLAG_SKIP_PRO,
  DEVPOD_FLAG_SOURCE,
  DEVPOD_FLAG_TIMEOUT,
  WORKSPACE_COMMAND_ADDITIONAL_FLAGS_KEY,
} from "../constants"

type TRawWorkspaces = readonly (Omit<TWorkspace, "status" | "id"> &
  Readonly<{ id: string | null }>)[]

export class WorkspaceCommands {
  static DEBUG = false
  static ADDITIONAL_FLAGS = new Map<string, string>()

  private static newCommand(args: string[]): Command {
    const extraFlags = []
    if (WorkspaceCommands.DEBUG) {
      extraFlags.push(DEVPOD_FLAG_DEBUG)
    }

    return new Command([...args, ...extraFlags])
  }

  static async ListWorkspaces(skipPro: boolean): Promise<Result<TWorkspaceWithoutStatus[]>> {
    const maybeSkipProFlag = skipPro ? [DEVPOD_FLAG_SKIP_PRO] : []

    const result = await new Command([
      DEVPOD_COMMAND_LIST,
      DEVPOD_FLAG_JSON_OUTPUT,
      ...maybeSkipProFlag,
    ]).run()
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

  static async GetWorkspaceUID() {
    const result = await new Command([
      DEVPOD_COMMAND_HELPER,
      DEVPOD_COMMAND_GET_WORKSPACE_UID,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return Return.Failed(`Failed to get UID: ${result.val.stderr}`)
    }

    return Return.Value(result.val.stdout)
  }

  static GetStatusLogs(id: string) {
    return new Command([DEVPOD_COMMAND_STATUS, id, DEVPOD_FLAG_JSON_LOG_OUTPUT])
  }

  static StartWorkspace(id: TWorkspaceID, config: TWorkspaceStartConfig) {
    const maybeSource = config.sourceConfig?.source
    const maybeIDFlag = exists(maybeSource) ? [toFlagArg(DEVPOD_FLAG_ID, id)] : []

    const maybeSourceType = config.sourceConfig?.type
    const maybeSourceFlag =
      exists(maybeSourceType) && exists(maybeSource)
        ? [toFlagArg(DEVPOD_FLAG_SOURCE, `${maybeSourceType}:${maybeSource}`)]
        : []
    const identifier = exists(maybeSource) && exists(maybeIDFlag) ? maybeSource : id

    const maybeIdeName = config.ideConfig?.name
    const maybeIDEFlag = exists(maybeIdeName) ? [toFlagArg(DEVPOD_FLAG_IDE, maybeIdeName)] : []

    const maybeProviderID = config.providerConfig?.providerID
    const maybeProviderFlag = exists(maybeProviderID)
      ? [toFlagArg(DEVPOD_FLAG_PROVIDER, maybeProviderID)]
      : []
    const maybeProviderOptions = config.providerConfig?.options
    const maybeProviderOptionsFlag = exists(maybeProviderOptions)
      ? serializeRawOptions(maybeProviderOptions, DEVPOD_FLAG_PROVIDER_OPTION)
      : []

    const maybePrebuildRepositories = config.prebuildRepositories?.length
      ? [toFlagArg(DEVPOD_FLAG_PREBUILD_REPOSITORY, config.prebuildRepositories.join(","))]
      : []

    const maybeDevcontainerPath = config.devcontainerPath
      ? [toFlagArg(DEVPOD_FLAG_DEVCONTAINER_PATH, config.devcontainerPath)]
      : []

    const additionalFlags = []
    if (WorkspaceCommands.ADDITIONAL_FLAGS.size > 0) {
      for (const [key, value] of WorkspaceCommands.ADDITIONAL_FLAGS.entries()) {
        if (key === WORKSPACE_COMMAND_ADDITIONAL_FLAGS_KEY) {
          additionalFlags.push(value)
          continue
        }

        additionalFlags.push(toFlagArg(key, value))
      }
    }

    return WorkspaceCommands.newCommand([
      DEVPOD_COMMAND_UP,
      identifier,
      ...maybeIDFlag,
      ...maybeSourceFlag,
      ...maybeIDEFlag,
      ...maybeProviderFlag,
      ...maybePrebuildRepositories,
      ...maybeDevcontainerPath,
      ...additionalFlags,
      ...maybeProviderOptionsFlag,
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

  static TroubleshootWorkspace(id: TWorkspaceID) {
    return WorkspaceCommands.newCommand([DEVPOD_COMMAND_TROUBLESHOOT, id])
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
