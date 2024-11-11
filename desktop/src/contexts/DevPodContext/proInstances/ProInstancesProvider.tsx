import { useQuery } from "@tanstack/react-query"
import { createContext, ReactNode, useMemo } from "react"
import { client } from "../../../client"
import { QueryKeys } from "../../../queryKeys"
import { TProInstances, TQueryResult } from "../../../types"
import { useChangeSettings } from "../../SettingsContext"
import { REFETCH_INTERVAL_MS } from "../constants"

export type TProInstancesContext = TQueryResult<TProInstances>
export const ProInstancesContext = createContext<TProInstancesContext>([
  [],
] as unknown as TProInstancesContext)

export function ProInstancesProvider({ children }: Readonly<{ children?: ReactNode }>) {
  const { set } = useChangeSettings()

  const proInstancesQuery = useQuery({
    queryKey: QueryKeys.PRO_INSTANCES,
    queryFn: async () => {
      const proInstances = (await client.pro.listProInstances({ authenticate: true })).unwrap()
      if (proInstances !== undefined && proInstances.length > 0) {
        set("experimental_devPodPro", true)
      }

      return proInstances
    },
    refetchInterval: REFETCH_INTERVAL_MS,
  })

  const value = useMemo<TProInstancesContext>(
    () => [
      proInstancesQuery.data,
      { status: proInstancesQuery.status, error: proInstancesQuery.error },
    ],
    [proInstancesQuery.data, proInstancesQuery.status, proInstancesQuery.error]
  )

  return <ProInstancesContext.Provider value={value}>{children}</ProInstancesContext.Provider>
}
