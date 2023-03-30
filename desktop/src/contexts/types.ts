/* eslint-disable @typescript-eslint/no-unused-vars */
import { ReactNode } from "react"
import { TStreamEventListenerFn } from "../client"
import { TWorkspace, TWorkspaceID } from "../types"

// TODO: remove file again
type TWorkspaceResult = Readonly<{
  data: TWorkspace | null
  loading: boolean
  current: Readonly<{
    action: TActionName | null
    connect: TConnectOperationFn
  }>
  history: Readonly<{
    get: (actionID: TActionID) => TAction
    getAll: (actionName: TActionName) => readonly TAction[]
  }>
  start: null
  create: null
  stop: null
  remove: null
  rebuild: null
}>

class WorkspaceResult {
  private _data: unknown

  public get data() {
    return this._data
  }

  constructor() {}

  function() {}
}

type TAction = Readonly<{
  id: TActionID
  name: TActionName
  stdout: readonly string[]
  stderr: readonly string[]
}>
type TActionID = string
type TActionName = keyof Pick<TWorkspaceResult, "create" | "start" | "stop" | "remove" | "rebuild">

declare function useWorkspace(workspaceID: TWorkspaceID): TWorkspaceResult
declare function useWorkspaces(): readonly TWorkspaceResult[]

type TConnectOperationFn = (connectConfig: Readonly<{ onStream?: TStreamEventListenerFn }>) => void
declare function useStreamingTerminal(): Readonly<{
  terminal: ReactNode
  connectStream: TStreamEventListenerFn
}>

type TStore = Record<TWorkspaceID, TWorkspaceResult>
