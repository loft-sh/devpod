import { Err, Failed } from "@/lib"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useMemo } from "react"
import { client } from "../../client"
import { QueryKeys } from "../../queryKeys"
import { TProInstanceLoginConfig, TProInstanceManager, TProvider, TWithProID } from "../../types"

export function useProInstanceManager(): TProInstanceManager {
  const queryClient = useQueryClient()
  const loginMutation = useMutation<TProvider | undefined, Error, TProInstanceLoginConfig>({
    mutationFn: async ({ host, providerName, streamListener }) => {
      ;(await client.pro.login(host, providerName, streamListener)).unwrap()

      try {
        const providers = (await client.providers.listAll()).unwrap()
        if (providers === undefined || Object.keys(providers).length === 0) {
          throw new Error("No providers found")
        }
        if (providerName === undefined || providerName === "") {
          providerName = "devpod-pro"
        }
        const maybeProvider = providers[providerName]
        if (!maybeProvider) {
          throw new Error(`Provider ${providerName} not found`)
        }

        return maybeProvider
      } catch (e) {
        ;(await client.pro.remove(host)).unwrap()

        throw e
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries(QueryKeys.PRO_INSTANCES)
      queryClient.invalidateQueries(QueryKeys.PROVIDERS)
    },
  })
  const disconnectMutation = useMutation<undefined, Err<Failed>, TWithProID>({
    mutationFn: async ({ id }) => (await client.pro.remove(id)).unwrap(),
    onSuccess: () => {
      queryClient.invalidateQueries(QueryKeys.PRO_INSTANCES)
      queryClient.invalidateQueries(QueryKeys.PROVIDERS)
    },
  })

  return useMemo(
    () => ({
      login: {
        run: loginMutation.mutate,
        status: loginMutation.status,
        error: loginMutation.error,
        reset: loginMutation.reset,
        provider: loginMutation.data,
      },
      disconnect: {
        run: disconnectMutation.mutate,
        status: disconnectMutation.status,
        error: disconnectMutation.error,
        target: disconnectMutation.variables,
      },
    }),
    [disconnectMutation, loginMutation]
  )
}
