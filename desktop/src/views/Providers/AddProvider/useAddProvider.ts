import { UseMutationOptions, useMutation, useQueryClient } from "@tanstack/react-query"
import { client } from "../../../client"
import { QueryKeys } from "../../../queryKeys"
import { TAddProviderConfig, TProviderID } from "../../../types"

type TAddUserMutationOptions = UseMutationOptions<
  Readonly<{
    providerID: TProviderID
    suggestedOptions: Record<string, string>
  }>,
  unknown,
  Readonly<{
    rawProviderSource: string
    config: TAddProviderConfig
  }>,
  unknown
>
type TUseAddProvider = Pick<TAddUserMutationOptions, "onSuccess" | "onError">
export function useAddProvider({ onSuccess, onError }: TUseAddProvider) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ rawProviderSource, config }) => {
      // check if provider exists and is not initialized
      const providerID = config.name || (await client.providers.newID(rawProviderSource)).unwrap()
      if (!providerID) {
        throw new Error(`Couldn't find provider id`)
      }

      // list all providers
      let providers = (await client.providers.listAll()).unwrap()
      if (providers?.[providerID]) {
        if (!providers[providerID]?.state?.initialized) {
          ;(await client.providers.remove(providerID)).unwrap()
        } else {
          throw new Error(
            `Provider with name ${providerID} already exists, please choose a different name`
          )
        }
      }

      // add provider
      ;(await client.providers.add(rawProviderSource, config)).unwrap()

      // check if provider could be added
      providers = (await client.providers.listAll()).unwrap()
      if (!providers?.[providerID!]) {
        throw new Error(`Provider ${providerID} couldn't be found`)
      }

      return {
        providerID: providerID!,
        suggestedOptions: {},
      }
    },
    onSuccess(result, ...rest) {
      queryClient.invalidateQueries(QueryKeys.PROVIDERS)
      onSuccess?.(result, ...rest)
    },
    onError,
  })
}
