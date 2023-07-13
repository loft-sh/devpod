import { Result, ResultError } from "../../lib"
import { TContextOptionName, TContextOptions } from "../../types"
import { TDebuggable } from "../types"
import { ContextCommands } from "./contextCommands"

export class ContextClient implements TDebuggable {
  constructor() {}

  public setDebug(isEnabled: boolean): void {
    ContextCommands.DEBUG = isEnabled
  }

  public async setOption(option: TContextOptionName, value: string): Promise<ResultError> {
    return ContextCommands.SetOptions({ [option]: value })
  }

  public async listOptions(): Promise<Result<TContextOptions>> {
    return ContextCommands.ListOptions()
  }
}
