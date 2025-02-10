import { Result, ResultError, Return, getErrorFromChildProcess } from "../../lib"
import { TContextOptionName, TContextOptions } from "../../types"
import { Command, isOk, serializeRawOptions } from "../command"
import {
  DEVPOD_COMMAND_CONTEXT,
  DEVPOD_COMMAND_OPTIONS,
  DEVPOD_COMMAND_SET_OPTIONS,
  DEVPOD_FLAG_DEBUG,
  DEVPOD_FLAG_JSON_LOG_OUTPUT,
  DEVPOD_FLAG_JSON_OUTPUT,
} from "../constants"

export class ContextCommands {
  static DEBUG = false

  private static newCommand(args: string[]): Command {
    return new Command([...args, ...(ContextCommands.DEBUG ? [DEVPOD_FLAG_DEBUG] : [])])
  }

  static async SetOptions(
    rawOptions: Partial<Record<TContextOptionName, string>>
  ): Promise<ResultError> {
    const optionsFlag = serializeRawOptions(rawOptions)
    const result = await ContextCommands.newCommand([
      DEVPOD_COMMAND_CONTEXT,
      DEVPOD_COMMAND_SET_OPTIONS,
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

  static async ListOptions(): Promise<Result<TContextOptions>> {
    const result = await ContextCommands.newCommand([
      DEVPOD_COMMAND_CONTEXT,
      DEVPOD_COMMAND_OPTIONS,
      DEVPOD_FLAG_JSON_OUTPUT,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    const options = JSON.parse(result.val.stdout) as TContextOptions

    return Return.Value(options)
  }
}
