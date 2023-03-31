import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useMemo } from "react"
import { client } from "../../client"
import { exists } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import { TProviderManager, TProviders, TWithProviderID } from "../../types"

export function useProviderManager(): TProviderManager {
  const queryClient = useQueryClient()

  const removeMutation = useMutation({
    mutationFn: async ({ providerID }: TWithProviderID) =>
      (await client.providers.remove(providerID)).unwrap(),
    onMutate({ providerID }) {
      // Optimistically updates `delete` mutation
      queryClient.cancelQueries(QueryKeys.PROVIDERS)
      const oldProviderSnapshot = queryClient.getQueryData<TProviders>(QueryKeys.PROVIDERS)?.[
        providerID
      ]
      queryClient.setQueryData<TProviders>(QueryKeys.PROVIDERS, (current) => {
        const shallowCopy = { ...current }
        delete shallowCopy[providerID]

        return shallowCopy
      })

      return { oldProviderSnapshot }
    },
    onError(_, { providerID }, ctx) {
      const maybeOldProvider = ctx?.oldProviderSnapshot
      if (exists(maybeOldProvider)) {
        queryClient.setQueryData<TProviders>(QueryKeys.PROVIDERS, (current) => ({
          ...current,
          [providerID]: maybeOldProvider,
        }))
      }
    },
    onSuccess(_, { providerID }) {
      queryClient.invalidateQueries(QueryKeys.provider(providerID))
    },
  })

  return useMemo(
    () => ({
      remove: {
        run: removeMutation.mutate,
        status: removeMutation.status,
        error: removeMutation.error,
        target: removeMutation.variables,
      },
    }),
    [removeMutation]
  )
}
