import { Command } from "../command"
import {
  DEVPOD_COMMAND_IDE,
  DEVPOD_COMMAND_LIST,
  DEVPOD_FLAG_DEBUG,
  DEVPOD_FLAG_JSON_OUTPUT,
} from "../constants"
import { Result, Return } from "../../lib"
import { TIDEs } from "../../types"

export class IDECommands {
  static DEBUG = false

  private static newCommand(args: string[]): Command {
    return new Command([...args, ...(IDECommands.DEBUG ? [DEVPOD_FLAG_DEBUG] : [])])
  }

  static async ListIDEs(): Promise<Result<TIDEs>> {
    const result = await IDECommands.newCommand([
      DEVPOD_COMMAND_IDE,
      DEVPOD_COMMAND_LIST,
      DEVPOD_FLAG_JSON_OUTPUT,
    ]).run()
    if (result.err) {
      return result
    }

    const ides = JSON.parse(result.val.stdout) as TIDEs

    return Return.Value(ides)
  }
}
