import { useContext, useMemo } from "react"
import { TWorkspaceManager } from "../../types"
import { DevPodContext, TDevpodContext } from "./DevPodProvider"
import { useWorkspaceManager } from "./useWorkspaceManager"

export function useWorkspaces(): [TDevpodContext["workspaces"], TWorkspaceManager] {
  const { workspaces } = useContext(DevPodContext)
  const manager = useWorkspaceManager()

  return useMemo(() => [workspaces, manager], [manager, workspaces])
}
