import { Result, ResultError, Return, getErrorFromChildProcess } from "@/lib"
import {
  TImportWorkspaceConfig,
  TListProInstancesConfig,
  TPlatformHealthCheck,
  TProID,
  TProInstance,
  TPlatformVersionInfo,
  TPlatformUpdateCheck,
} from "@/types"
import { Command, isOk, serializeRawOptions, toFlagArg } from "../command"
import {
  DEVPOD_COMMAND_DELETE,
  DEVPOD_COMMAND_IMPORT_WORKSPACE,
  DEVPOD_COMMAND_LIST,
  DEVPOD_COMMAND_LOGIN,
  DEVPOD_COMMAND_PRO,
  DEVPOD_FLAG_ACCESS_KEY,
  DEVPOD_FLAG_DEBUG,
  DEVPOD_FLAG_FORCE_BROWSER,
  DEVPOD_FLAG_HOST,
  DEVPOD_FLAG_INSTANCE,
  DEVPOD_FLAG_JSON_LOG_OUTPUT,
  DEVPOD_FLAG_JSON_OUTPUT,
  DEVPOD_FLAG_LOGIN,
  DEVPOD_FLAG_PROJECT,
  DEVPOD_FLAG_USE,
  DEVPOD_FLAG_WORKSPACE_ID,
  DEVPOD_FLAG_WORKSPACE_PROJECT,
  DEVPOD_FLAG_WORKSPACE_UID,
} from "../constants"
import { TStreamEventListenerFn } from "../types"
import { ManagementV1DevPodWorkspaceInstance } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceInstance"
import { ManagementV1Project } from "@loft-enterprise/client/gen/models/managementV1Project"
import { ManagementV1Self } from "@loft-enterprise/client/gen/models/managementV1Self"
import { ManagementV1ProjectTemplates } from "@loft-enterprise/client/gen/models/managementV1ProjectTemplates"
import { ManagementV1ProjectClusters } from "@loft-enterprise/client/gen/models/managementV1ProjectClusters"

export class ProCommands {
  static DEBUG = false

  private static newCommand(args: string[]): Command {
    return new Command([...args, ...(ProCommands.DEBUG ? [DEVPOD_FLAG_DEBUG] : [])])
  }

  static async Login(
    host: string,
    accessKey?: string,
    listener?: TStreamEventListenerFn
  ): Promise<ResultError> {
    const maybeAccessKeyFlag = accessKey ? [toFlagArg(DEVPOD_FLAG_ACCESS_KEY, accessKey)] : []
    const useFlag = toFlagArg(DEVPOD_FLAG_USE, "false")

    const cmd = ProCommands.newCommand([
      DEVPOD_COMMAND_PRO,
      DEVPOD_COMMAND_LOGIN,
      host,
      useFlag,
      DEVPOD_FLAG_FORCE_BROWSER,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
      ...maybeAccessKeyFlag,
    ])
    if (listener) {
      return cmd.stream(listener)
    } else {
      const result = await cmd.run()
      if (result.err) {
        return result
      }

      if (!isOk(result.val)) {
        return getErrorFromChildProcess(result.val)
      }

      return Return.Ok()
    }
  }

  static async ListProInstances(
    config?: TListProInstancesConfig
  ): Promise<Result<readonly TProInstance[]>> {
    const maybeLoginFlag = config?.authenticate ? [DEVPOD_FLAG_LOGIN] : []
    const result = await ProCommands.newCommand([
      DEVPOD_COMMAND_PRO,
      DEVPOD_COMMAND_LIST,
      DEVPOD_FLAG_JSON_OUTPUT,
      ...maybeLoginFlag,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    const instances = JSON.parse(result.val.stdout) as readonly TProInstance[]

    return Return.Value(instances)
  }

  static async RemoveProInstance(id: TProID) {
    const result = await ProCommands.newCommand([
      DEVPOD_COMMAND_PRO,
      DEVPOD_COMMAND_DELETE,
      id,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Ok()
  }

  static async ImportWorkspace(config: TImportWorkspaceConfig): Promise<ResultError> {
    const optionsFlag = config.options ? serializeRawOptions(config.options) : []
    const result = await new Command([
      DEVPOD_COMMAND_PRO,
      DEVPOD_COMMAND_IMPORT_WORKSPACE,
      config.devPodProHost,
      DEVPOD_FLAG_WORKSPACE_ID,
      config.workspaceID,
      DEVPOD_FLAG_WORKSPACE_UID,
      config.workspaceUID,
      DEVPOD_FLAG_WORKSPACE_PROJECT,
      config.project,
      ...optionsFlag,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Ok()
  }

  static WatchWorkspaces(id: TProID, projectName: string) {
    const hostFlag = toFlagArg(DEVPOD_FLAG_HOST, id)
    const projectFlag = toFlagArg(DEVPOD_FLAG_PROJECT, projectName)
    const args = [DEVPOD_COMMAND_PRO, "watch-workspaces", hostFlag, projectFlag]

    return ProCommands.newCommand(args)
  }

  static async ListProjects(id: TProID) {
    const hostFlag = toFlagArg(DEVPOD_FLAG_HOST, id)
    const args = [DEVPOD_COMMAND_PRO, "list-projects", hostFlag]

    const result = await ProCommands.newCommand(args).run()
    if (result.err) {
      return result
    }
    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as readonly ManagementV1Project[])
  }

  static async GetSelf(id: TProID) {
    const hostFlag = toFlagArg(DEVPOD_FLAG_HOST, id)
    const args = [DEVPOD_COMMAND_PRO, "self", hostFlag]

    const result = await ProCommands.newCommand(args).run()
    if (result.err) {
      return result
    }
    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as ManagementV1Self)
  }

  static async ListTemplates(id: TProID, projectName: string) {
    const hostFlag = toFlagArg(DEVPOD_FLAG_HOST, id)
    const projectFlag = toFlagArg(DEVPOD_FLAG_PROJECT, projectName)
    const args = [DEVPOD_COMMAND_PRO, "list-templates", hostFlag, projectFlag]

    const result = await ProCommands.newCommand(args).run()
    if (result.err) {
      return result
    }
    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as ManagementV1ProjectTemplates)
  }

  static async ListClusters(id: TProID, projectName: string) {
    const hostFlag = toFlagArg(DEVPOD_FLAG_HOST, id)
    const projectFlag = toFlagArg(DEVPOD_FLAG_PROJECT, projectName)
    const args = [DEVPOD_COMMAND_PRO, "list-clusters", hostFlag, projectFlag]

    const result = await ProCommands.newCommand(args).run()
    if (result.err) {
      return result
    }
    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as ManagementV1ProjectClusters)
  }

  static async CreateWorkspace(id: TProID, instance: ManagementV1DevPodWorkspaceInstance) {
    const hostFlag = toFlagArg(DEVPOD_FLAG_HOST, id)
    const instanceFlag = toFlagArg(DEVPOD_FLAG_INSTANCE, JSON.stringify(instance))
    const args = [DEVPOD_COMMAND_PRO, "create-workspace", hostFlag, instanceFlag]

    const result = await ProCommands.newCommand(args).run()
    if (result.err) {
      return result
    }
    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as ManagementV1DevPodWorkspaceInstance)
  }

  static async UpdateWorkspace(id: TProID, instance: ManagementV1DevPodWorkspaceInstance) {
    const hostFlag = toFlagArg(DEVPOD_FLAG_HOST, id)
    const instanceFlag = toFlagArg(DEVPOD_FLAG_INSTANCE, JSON.stringify(instance))
    const args = [DEVPOD_COMMAND_PRO, "update-workspace", hostFlag, instanceFlag]

    const result = await ProCommands.newCommand(args).run()
    if (result.err) {
      return result
    }
    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as ManagementV1DevPodWorkspaceInstance)
  }

  static async CheckHealth(id: TProID) {
    const hostFlag = toFlagArg(DEVPOD_FLAG_HOST, id)
    const args = [DEVPOD_COMMAND_PRO, "check-health", hostFlag]

    const result = await ProCommands.newCommand(args).run()
    if (result.err) {
      return result
    }
    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as TPlatformHealthCheck)
  }

  static async GetVersion(id: TProID) {
    const hostFlag = toFlagArg(DEVPOD_FLAG_HOST, id)
    const args = [DEVPOD_COMMAND_PRO, "version", hostFlag]

    const result = await ProCommands.newCommand(args).run()
    if (result.err) {
      return result
    }
    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as TPlatformVersionInfo)
  }

  static async CheckUpdate(id: TProID) {
    const hostFlag = toFlagArg(DEVPOD_FLAG_HOST, id)
    const args = [DEVPOD_COMMAND_PRO, "check-update", hostFlag]

    const result = await ProCommands.newCommand(args).run()
    if (result.err) {
      return result
    }
    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as TPlatformUpdateCheck)
  }

  static async Update(id: TProID, version: string) {
    const hostFlag = toFlagArg(DEVPOD_FLAG_HOST, id)
    const args = [DEVPOD_COMMAND_PRO, "update-provider", version, hostFlag]

    const result = await ProCommands.newCommand(args).run()
    if (result.err) {
      return result
    }
    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as TPlatformUpdateCheck)
  }
}
