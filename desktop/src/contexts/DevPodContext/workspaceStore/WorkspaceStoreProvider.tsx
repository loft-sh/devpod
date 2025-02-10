import { createContext, ReactNode, useMemo } from "react"
import { IWorkspaceStore } from "./workspaceStore"

export type TWorkspaceStoreContext = Readonly<{
  store: IWorkspaceStore<string, unknown>
}>
export const WorkspaceStoreContext = createContext<TWorkspaceStoreContext>(null!)

type TWorkspaceStoreProps<TStore> = Readonly<{
  store: TStore
  children?: ReactNode
}>
export function WorkspaceStoreProvider<TStore extends IWorkspaceStore<string, any>>({
  children,
  store,
}: TWorkspaceStoreProps<TStore>) {
  const value = useMemo(() => ({ store }), [store])

  return <WorkspaceStoreContext.Provider value={value}>{children}</WorkspaceStoreContext.Provider>
}
