import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useMemo } from "react"
import { client } from "../../client"
import { QueryKeys } from "../../queryKeys"
import { TProviderManagerRunConfig, TProviders } from "../../types"
import { getOperationManagerFromMutation } from "./helpers"

export function useProviderManager() {
  const queryClient = useQueryClient()

  const removeMutation = useMutation({
    mutationFn: ({ providerID }: TProviderManagerRunConfig["remove"]) =>
      client.providers.remove(providerID),
    onSuccess(_, { providerID }) {
      queryClient.setQueryData<TProviders>(QueryKeys.PROVIDERS, (current) => {
        const shallowCopy = { ...current }
        delete shallowCopy[providerID]

        return shallowCopy
      })
    },
  })

  return useMemo(
    () => ({
      remove: getOperationManagerFromMutation(removeMutation),
    }),
    [removeMutation]
  )
}
