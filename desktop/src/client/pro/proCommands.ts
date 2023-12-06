import { Result, ResultError, Return, getErrorFromChildProcess } from "@/lib"
import { TImportWorkspaceConfig, TProID, TProInstance } from "@/types"
import { Command, isOk, serializeRawOptions, toFlagArg } from "../command"
import {
  DEVPOD_COMMAND_DELETE,
  DEVPOD_COMMAND_IMPORT_WORKSPACE,
  DEVPOD_COMMAND_LIST,
  DEVPOD_COMMAND_LOGIN,
  DEVPOD_COMMAND_PRO,
  DEVPOD_FLAG_ACCESS_KEY,
  DEVPOD_FLAG_DEBUG,
  DEVPOD_FLAG_JSON_LOG_OUTPUT,
  DEVPOD_FLAG_JSON_OUTPUT,
  DEVPOD_FLAG_PROVIDER,
  DEVPOD_FLAG_USE,
  DEVPOD_FLAG_WORKSPACE_ID,
  DEVPOD_FLAG_WORKSPACE_UID,
} from "../constants"
import { TStreamEventListenerFn } from "../types"

export class ProCommands {
  static DEBUG = false

  private static newCommand(args: string[]): Command {
    return new Command([...args, ...(ProCommands.DEBUG ? [DEVPOD_FLAG_DEBUG] : [])])
  }

  static async Login(
    host: string,
    providerName?: string,
    accessKey?: string,
    listener?: TStreamEventListenerFn
  ): Promise<ResultError> {
    const maybeProviderNameFlag = providerName
      ? [toFlagArg(DEVPOD_FLAG_PROVIDER, providerName)]
      : []
    const maybeAccessKeyFlag = accessKey ? [toFlagArg(DEVPOD_FLAG_ACCESS_KEY, accessKey)] : []
    const useFlag = toFlagArg(DEVPOD_FLAG_USE, "false")

    const cmd = ProCommands.newCommand([
      DEVPOD_COMMAND_PRO,
      DEVPOD_COMMAND_LOGIN,
      host,
      useFlag,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
      ...maybeProviderNameFlag,
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

  static async ListProInstances(): Promise<Result<readonly TProInstance[]>> {
    const result = await ProCommands.newCommand([
      DEVPOD_COMMAND_PRO,
      DEVPOD_COMMAND_LIST,
      DEVPOD_FLAG_JSON_OUTPUT,
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
}
