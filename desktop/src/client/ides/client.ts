import { TDebuggable } from "../types"
import { Result, ResultError } from "../../lib"
import { TIDEs } from "../../types"
import { IDECommands } from "./ideCommands"

export class IDEsClient implements TDebuggable {
  constructor() {}

  public setDebug(isEnabled: boolean): void {
    IDECommands.DEBUG = isEnabled
  }

  public async useIDE(ide: string): Promise<ResultError> {
    return IDECommands.UseIDE(ide)
  }

  public async listAll(): Promise<Result<TIDEs>> {
    return IDECommands.ListIDEs()
  }
}
