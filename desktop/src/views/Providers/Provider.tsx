import { Box, Spinner } from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import { useMemo } from "react"
import { useNavigate, useParams } from "react-router"
import { client } from "../../client"
import { useProvider } from "../../contexts"
import { exists } from "../../lib"
import { Routes } from "../../routes"
import { ConfigureProviderOptionsForm } from "./AddProvider/ConfigureProviderOptionsForm"
import { QueryKeys } from "../../queryKeys"

export function Provider() {
  const navigate = useNavigate()
  const params = useParams()
  const providerID = useMemo(() => Routes.getProviderId(params), [params])
  const [provider] = useProvider(providerID)
  const providerOptionsQuery = useQuery({
    queryKey: [QueryKeys.PROVIDERS, providerID],
    queryFn: async () => (await client.providers.getOptions(providerID!)).unwrap(),
  })

  if (!exists(provider)) {
    return <Spinner />
  }

  if (!exists(providerID)) {
    return null
  }

  return (
    <Box width="full">
      <ConfigureProviderOptionsForm
        providerID={providerID}
        isDefault={!!provider.default}
        addProvider={false}
        reuseMachine={!!provider.state?.singleMachine}
        options={providerOptionsQuery.data ?? {}}
        optionGroups={provider.config?.optionGroups || []}
        onFinish={() => navigate(Routes.PROVIDERS)}
      />
    </Box>
  )
}
