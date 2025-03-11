import { createContext } from "react"
import { TProviders, TQueryResult } from "../../../types"

export type TDevpodContext = Readonly<{
  providers: TQueryResult<TProviders>
}>
export const DevPodContext = createContext<TDevpodContext | null>(null)
