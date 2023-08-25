import { Result, ResultError } from "../../lib"
import {TImportWorkspaceConfig, TProID, TProInstance} from "../../types"
import { TDebuggable, TStreamEventListenerFn } from "../types"
import { ProCommands } from "./proCommands"
import {ProviderCommands} from "@/client/providers/providerCommands";

export class ProClient implements TDebuggable {
  constructor() {}

  public setDebug(isEnabled: boolean): void {
    ProCommands.DEBUG = isEnabled
  }

  public async newID(url: string): Promise<Result<string>> {
    return ProCommands.GetProInstanceID(url)
  }

  public async login(
    url: string,
    name?: string,
    listener?: TStreamEventListenerFn
  ): Promise<ResultError> {
    return ProCommands.Login(url, name, listener)
  }

  public async listAll(): Promise<Result<readonly TProInstance[]>> {
    return ProCommands.ListProInstances()
  }

  public async remove(id: TProID) {
    return ProCommands.RemoveProInstance(id)
  }

  public async importWorkspace(config: TImportWorkspaceConfig): Promise<ResultError> {
    return ProCommands.ImportWorkspace(config)
  }
}
