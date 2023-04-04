import { TDebuggable } from "../types"
import { Result } from "../../lib"
import { TIDEs, TProviderID } from "../../types"
import { IDECommands } from "./ideCommands"

export class IDEsClient implements TDebuggable {
  private danglingProviderID: TProviderID | null = null
  // Queues store operations and guarantees they will be executed in order
  private storeOperationQueue: Promise<unknown> = Promise.resolve()

  constructor() {}

  public setDebug(isEnabled: boolean): void {
    IDECommands.DEBUG = isEnabled
  }

  public async listAll(): Promise<Result<TIDEs>> {
    return IDECommands.ListIDEs()
  }
}
