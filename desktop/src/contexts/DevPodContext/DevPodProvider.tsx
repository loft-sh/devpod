import { useQuery } from "@tanstack/react-query"
import { createContext, ReactNode, useMemo } from "react"
import { client } from "../../client"
import { QueryKeys } from "../../queryKeys"
import { TProviders, TQueryResult } from "../../types"
import { REFETCH_PROVIDER_INTERVAL_MS } from "./constants"
import { usePollWorkspaces } from "./workspaces"

export type TDevpodContext = Readonly<{
  providers: TQueryResult<TProviders>
}>
export const DevPodContext = createContext<TDevpodContext>(null!)

export function DevPodProvider({ children }: Readonly<{ children?: ReactNode }>) {
  usePollWorkspaces()

  const providersQuery = useQuery({
    queryKey: QueryKeys.PROVIDERS,
    queryFn: async () => (await client.providers.listAll()).unwrap(),
    refetchInterval: REFETCH_PROVIDER_INTERVAL_MS,
    enabled: true,
  })

  const value = useMemo<TDevpodContext>(
    () => ({
      providers: [
        providersQuery.data,
        { status: providersQuery.status, error: providersQuery.error },
      ],
    }),
    [providersQuery.data, providersQuery.status, providersQuery.error]
  )

  return <DevPodContext.Provider value={value}>{children}</DevPodContext.Provider>
}
