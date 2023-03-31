import { useQuery, useQueryClient } from "@tanstack/react-query"
import { exists } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import { TProvider, TProviderID, TProviderManager, TProviders, TQueryResult } from "../../types"
import { useProviderManager } from "./useProviderManager"

export function useProvider(
  providerID: TProviderID | undefined | null
): [TQueryResult<TProvider>, TProviderManager] {
  const queryClient = useQueryClient()
  const manager = useProviderManager()

  const { data, status, error } = useQuery({
    queryKey: QueryKeys.provider(providerID!),
    queryFn: ({ queryKey }) => {
      const [maybeProviderKey] = queryKey

      if (!exists(maybeProviderKey)) {
        throw Error(`${maybeProviderKey} not found`)
      }

      const maybeProvider = Object.entries(
        queryClient.getQueryData<TProviders>([maybeProviderKey]) ?? {}
      ).find(([name]) => name === providerID)?.[1]

      if (!exists(maybeProvider)) {
        throw Error(`Provider ${providerID} not found`)
      }

      return maybeProvider
    },
    enabled: exists(providerID),
  })

  return [[data, { status, error }], manager]
}
