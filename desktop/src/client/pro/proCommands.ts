import { Result, ResultError, Return, getErrorFromChildProcess } from "@/lib"
import { TProID, TProInstance } from "@/types"
import { Command, isOk, toFlagArg } from "../command"
import {
  DEVPOD_COMMAND_DELETE,
  DEVPOD_COMMAND_GET_PRO_NAME,
  DEVPOD_COMMAND_HELPER,
  DEVPOD_COMMAND_LIST,
  DEVPOD_COMMAND_LOGIN,
  DEVPOD_COMMAND_PRO,
  DEVPOD_FLAG_DEBUG,
  DEVPOD_FLAG_JSON_LOG_OUTPUT,
  DEVPOD_FLAG_JSON_OUTPUT,
  DEVPOD_FLAG_NAME,
  DEVPOD_FLAG_USE,
} from "../constants"
import { TStreamEventListenerFn } from "../types"

export class ProCommands {
  static DEBUG = false

  private static newCommand(args: string[]): Command {
    return new Command([...args, ...(ProCommands.DEBUG ? [DEVPOD_FLAG_DEBUG] : [])])
  }

  static async GetProInstanceID(url: string) {
    const result = await new Command([
      DEVPOD_COMMAND_HELPER,
      DEVPOD_COMMAND_GET_PRO_NAME,
      url,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(result.val.stdout)
  }

  static async Login(
    url: string,
    name?: string,
    listener?: TStreamEventListenerFn
  ): Promise<ResultError> {
    const maybeNameFlag = name ? [toFlagArg(DEVPOD_FLAG_NAME, name)] : []
    const useFlag = toFlagArg(DEVPOD_FLAG_USE, "false")

    const cmd = await ProCommands.newCommand([
      DEVPOD_COMMAND_PRO,
      DEVPOD_COMMAND_LOGIN,
      url,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
      useFlag,
      ...maybeNameFlag,
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
}
