import { useContext, useMemo } from "react"
import { DevPodContext, TDevpodContext } from "./DevPodProvider"

export function useWorkspaces(): TDevpodContext["workspaces"] {
  const { workspaces } = useContext(DevPodContext)

  return useMemo(() => workspaces, [workspaces])
}
