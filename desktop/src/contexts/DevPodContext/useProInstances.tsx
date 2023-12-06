import { useContext } from "react"
import { TProInstanceManager } from "../../types"
import { DevPodContext, TDevpodContext } from "./DevPodProvider"
import { useProInstanceManager } from "./useProInstanceManager"

export function useProInstances(): [TDevpodContext["proInstances"], TProInstanceManager] {
  const proInstances = useContext(DevPodContext).proInstances
  const manager = useProInstanceManager()

  return [proInstances, manager]
}
