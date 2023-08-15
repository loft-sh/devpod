import { Err, Failed } from "@/lib"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useMemo } from "react"
import { client } from "../../client"
import { QueryKeys } from "../../queryKeys"
import { TProInstanceLoginConfig, TProInstanceManager, TProvider, TWithProID } from "../../types"

export function useProInstanceManager(): TProInstanceManager {
  const queryClient = useQueryClient()
  const loginMutation = useMutation<TProvider | undefined, Error, TProInstanceLoginConfig>({
    mutationFn: async ({ url, name, streamListener }) => {
      if (!name) {
        name = (await client.pro.newID(url)).unwrap()
      }
      if (!name) {
        throw new Error("No name provided")
      }

      ;(await client.pro.login(url, name, streamListener)).unwrap()

      try {
        const providers = (await client.providers.listAll()).unwrap()
        if (providers === undefined || Object.keys(providers).length === 0) {
          throw new Error("No providers found")
        }

        const maybeProvider = providers[name]
        if (!maybeProvider) {
          throw new Error(`Provider ${name} not found`)
        }

        return maybeProvider
      } catch (e) {
        ;(await client.pro.remove(name)).unwrap()

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
