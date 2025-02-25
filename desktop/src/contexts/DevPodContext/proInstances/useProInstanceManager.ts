import { DaemonClient } from "@/client/pro/client"
import { Err, Failed } from "@/lib"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useMemo } from "react"
import { client } from "../../../client"
import { QueryKeys } from "../../../queryKeys"
import { TProInstanceLoginConfig, TProInstanceManager, TProvider, TWithProID } from "../../../types"

const FALLBACK_PROVIDER_NAME = "devpod-pro"

export function useProInstanceManager(): TProInstanceManager {
  const queryClient = useQueryClient()
  const loginMutation = useMutation<TProvider | undefined, Error, TProInstanceLoginConfig>({
    mutationFn: async ({ host, accessKey, streamListener }) => {
      ;(await client.pro.login(host, accessKey, streamListener)).unwrap()

      // if we don't have a provider name, check for the pro instance and then use it's provider name
      const proInstances = (await client.pro.listProInstances()).unwrap()
      const maybeNewInstance = proInstances?.find((instance) => instance.host === host)
      let maybeProviderName = maybeNewInstance?.provider

      if (maybeNewInstance) {
        const proClient = client.getProClient(maybeNewInstance)
        if (proClient instanceof DaemonClient) {
          await proClient.restartDaemon()
        }
      }
      try {
        const providers = (await client.providers.listAll()).unwrap()
        if (providers === undefined || Object.keys(providers).length === 0) {
          throw new Error("No providers found")
        }
        if (!maybeProviderName) {
          maybeProviderName = FALLBACK_PROVIDER_NAME
        }
        const maybeProvider = providers[maybeProviderName]
        if (!maybeProvider) {
          throw new Error(`Provider ${maybeProviderName} not found`)
        }

        return maybeProvider
      } catch (e) {
        ;(await client.pro.removeProInstance(host)).unwrap()

        throw e
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries(QueryKeys.PRO_INSTANCES)
      queryClient.invalidateQueries(QueryKeys.PROVIDERS)
    },
  })
  const disconnectMutation = useMutation<undefined, Err<Failed>, TWithProID>({
    mutationFn: async ({ id }) => (await client.pro.removeProInstance(id)).unwrap(),
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
