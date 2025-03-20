import { createContext } from "react"
import { IWorkspaceStore } from "./workspaceStore"

export type TWorkspaceStoreContext = Readonly<{
  store: IWorkspaceStore<string, unknown>
}>
export const WorkspaceStoreContext = createContext<TWorkspaceStoreContext>(null!)
