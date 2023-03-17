// TODO: types need some love :)

import { UseMutationResult } from "@tanstack/react-query"
import { TStreamEventHandlerFn } from "./client"

export type TUnsubscribeFn = VoidFunction
export type TLogOutput = Readonly<{ time: Date; message: string; level: string }>
export type TProviderID = string
export type TProviders = Readonly<{
  defaultProvider: string
  providers: Record<TProviderID, { options: Record<string, TOption> | null }>
}>
export type TOption = Readonly<{
  local: string | null
  retrieved: string | null
  value: string | null
}>

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

type TRunnable<TRunConfig = TWithWorkspaceID> = Readonly<{ run(config: TRunConfig): void }>
type TOperationStatus = Pick<UseMutationResult, "status" | "error">
export type TWithWorkspaceID = Readonly<{ workspaceID: TWorkspaceID }>
export type TOperationManager<TRunConfig = TWithWorkspaceID> = TRunnable<TRunConfig> &
  TOperationStatus

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
      onStream?: TStreamEventHandlerFn
    }>
  >
  start: TOperationManager<
    TWithWorkspaceID & Readonly<{ config: TWorkspaceStartConfig; onStream?: TStreamEventHandlerFn }>
  >
  stop: TOperationManager
  remove: TOperationManager
  rebuild: TOperationManager
}>
export type TWorkspaceManagerRunConfig<T extends keyof TWorkspaceManager> = Parameters<
  TWorkspaceManager[T]["run"]
>[0]

export type TQueryResult<TData extends Readonly<object>> = [TData | undefined, TOperationStatus]
