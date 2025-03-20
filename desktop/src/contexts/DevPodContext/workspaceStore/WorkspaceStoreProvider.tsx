import { ReactNode, useMemo } from "react"
import { WorkspaceStoreContext } from "./context"
import { IWorkspaceStore } from "./workspaceStore"

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
