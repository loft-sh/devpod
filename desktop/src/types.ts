import { UseMutationResult } from "@tanstack/react-query"
import { TStreamEventListenerFn } from "./client"

export type TUnsubscribeFn = VoidFunction
export type TComparable<T> = Readonly<{ eq(b: T): boolean }>
export type TIdentifiable = Readonly<{ id: string }>
export type TViewID = string

//#region Shared
export type TLogOutput = Readonly<{ time: Date; message: string; level: string }>
export type TQueryResult<TData extends Readonly<object>> = [
  TData | undefined,
  TOperationStatus<unknown>
]
export type TConnectConfig<T extends TWithResourceID> = T & TWithStream
export type TConnectOperationFn<T extends TWithResourceID> = (
  connectConfig: TConnectConfig<T>
) => void
export type TOperationManager<TRunConfig = TWithResourceID> = TRunnable<TRunConfig> &
  TOperationStatus<TRunConfig>
export type TOperationManagerRunConfig<
  TManager extends Record<string, TOperationManager<unknown>>
> = {
  [K in keyof TManager]: Parameters<TManager[K]["run"]>[0]
}
type TOperationStatus<TRunConfig> = Pick<UseMutationResult, "status" | "error"> &
  Readonly<{ target: UseMutationResult<unknown, unknown, TRunConfig>["variables"] }>
type TWithStream = Readonly<{ onStream?: TStreamEventListenerFn }>
type TRunnable<TRunConfig = TWithResourceID> = Readonly<{ run(config: TRunConfig): void }>
type TConnectable<T extends TWithResourceID> = Readonly<{ connect: TConnectOperationFn<T> }>
//#endregion

//#region Provider
export type TProviderID = string
export type TWithProviderID = Readonly<{ providerID: TProviderID }>
export type TProviders = Record<TProviderID, TProvider>
export type TProvider = Readonly<{
  config: TProviderConfig | null
  state: null /* TODO: TYPE ME */
}>
export type TProviderConfig = Readonly<{
  name: string | null
  version: string | null
  source: TProviderSource | null
  description: string | null
  options: TProviderOptions
}>
type TProviderSource = Readonly<{
  github: string | null
  file: string | null
  url: string | null
}>
export type TProviderOptions = Record<string, TProviderOption>
export type TProviderOption = Readonly<{
  // TODO: add more stuff from go
  local: string | null
  retrieved: string | null
  value: string | null
}>
export type TAddProviderConfig = Readonly<{
  name?: TProviderConfig["name"]
}>
export type TProviderManager = Readonly<{
  remove: TOperationManager<TWithProviderID>
}>
export type TProviderManagerRunConfig = TOperationManagerRunConfig<TProviderManager>
//#endregion

//#region Workspace
export type TWorkspaceID = NonNullable<TWorkspace["id"]>
export type TWorkspace = Readonly<{
  id: string
  provider: Readonly<{ name: string | null }> | null
  status: "Running" | "Busy" | "Stopped" | "NotFound" | null
  ide: {
    ide: string | null
  } | null
  source: {
    gitRepository: string | null
    gitBranch: string | null
    localFolder: string | null
    image: string | null
  } | null
}>
export type TWorkspaceWithoutStatus = Omit<TWorkspace, "status"> & Readonly<{ status: null }>
export type TWorkspaceStatusResult = Readonly<{
  id: string | null
  context: string | null
  provider: string | null
  state: TWorkspace["status"] | null
}>
export type TWorkspaces = readonly TWorkspace[]
export type TWithWorkspaceID = Readonly<{ workspaceID: TWorkspaceID }>
export type TWithResourceID = TWithProviderID | TWithWorkspaceID
export type TWorkspaceStartConfig = Readonly<{
  ideConfig?: TWorkspace["ide"]
  providerConfig?: Readonly<{ providerID?: TProviderID }>
  // Instead of starting a workspace just by ID, the sourceConfig starts it with a `source/ID` combination
  sourceConfig?: Readonly<{
    source: string
  }>
}>
export type TWorkspaceManager = Readonly<{
  create: TOperationManager<
    Readonly<{
      rawWorkspaceSource: string
      config: TWorkspaceStartConfig
    }> &
      TWithStream
  >
  start: TOperationManager<
    TWithWorkspaceID &
      Readonly<{ config: TWorkspaceStartConfig; onStream?: TStreamEventListenerFn }>
  > &
    TConnectable<TWithWorkspaceID>
  stop: TOperationManager<TWithWorkspaceID>
  remove: TOperationManager<TWithWorkspaceID>
  rebuild: TOperationManager<TWithWorkspaceID>
}>
export type TWorkspaceManagerRunConfig = TOperationManagerRunConfig<TWorkspaceManager>
//#endregion
