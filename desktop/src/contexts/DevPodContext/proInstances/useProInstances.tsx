import { useContext } from "react"
import { TProInstanceManager } from "../../../types"
import { useProInstanceManager } from "./useProInstanceManager"
import { ProInstancesContext, TProInstancesContext } from "./ProInstancesProvider"

export function useProInstances(): [TProInstancesContext, TProInstanceManager] {
  const proInstances = useContext(ProInstancesContext)
  const manager = useProInstanceManager()

  return [proInstances, manager]
}
