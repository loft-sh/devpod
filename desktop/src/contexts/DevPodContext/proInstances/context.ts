import { createContext } from "react"
import { TProInstances, TQueryResult } from "../../../types"

export type TProInstancesContext = TQueryResult<TProInstances>
export const ProInstancesContext = createContext<TProInstancesContext>([
  [],
] as unknown as TProInstancesContext)
