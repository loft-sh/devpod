import { exists, Result, Return } from "../../lib"
import {
  TWorkspace,
  TWorkspaceID,
  TWorkspaceStartConfig,
  TWorkspaceStatusResult,
  TWorkspaceWithoutStatus,
} from "../../types"
import { Command, isOk, toFlagArg } from "../command"
import {
  DEVPOD_COMMAND_DELETE,
  DEVPOD_COMMAND_GET_WORKSPACE_NAME,
  DEVPOD_COMMAND_HELPER,
  DEVPOD_COMMAND_LIST,
  DEVPOD_COMMAND_STATUS,
  DEVPOD_COMMAND_STOP,
  DEVPOD_COMMAND_UP,
  DEVPOD_FLAG_DEBUG,
  DEVPOD_FLAG_FORCE_BUILD,
  DEVPOD_FLAG_ID,
  DEVPOD_FLAG_IDE,
  DEVPOD_FLAG_JSON_LOG_OUTPUT,
  DEVPOD_FLAG_JSON_OUTPUT,
  DEVPOD_FLAG_PREBUILD_REPOSITORY,
  DEVPOD_FLAG_PROVIDER,
  DEVPOD_FLAG_RECREATE,
} from "../constants"

type TRawWorkspaces = readonly (Omit<TWorkspace, "status" | "id"> &
  Readonly<{ id: string | null }>)[]

export class WorkspaceCommands {
  static DEBUG = false

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

  static async GetWorkspaceStatus(id: string): Promise<Result<Pick<TWorkspace, "id" | "status">>> {
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

    const identifier = exists(maybeSource) && exists(maybeIDFlag) ? maybeSource : id

    return WorkspaceCommands.newCommand([
      DEVPOD_COMMAND_UP,
      identifier,
      ...maybeIDFlag,
      ...maybeIDEFlag,
      ...maybeProviderFlag,
      ...maybePrebuildRepositories,
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

  static RemoveWorkspace(id: TWorkspaceID) {
    return WorkspaceCommands.newCommand([DEVPOD_COMMAND_DELETE, id, DEVPOD_FLAG_JSON_LOG_OUTPUT])
  }
}
