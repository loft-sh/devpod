import { useContext } from "react"
import { WorkspaceStoreContext } from "./context"
import { IWorkspaceStore } from "./workspaceStore"

export function useWorkspaceStore<T extends IWorkspaceStore<string, unknown>>() {
  const { store } = useContext(WorkspaceStoreContext)

  return { store: store as T }
}
