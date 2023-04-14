import { Box, Spinner, Text } from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import { useMemo } from "react"
import { useNavigate, useParams } from "react-router"
import { client } from "../../client"
import { ErrorMessageBox } from "../../components"
import { useProvider } from "../../contexts"
import { exists, isError } from "../../lib"
import { Routes } from "../../routes"
import { ConfigureProviderOptionsForm } from "./AddProvider/ConfigureProviderOptionsForm"
import { QueryKeys } from "../../queryKeys"

export function Provider() {
  const navigate = useNavigate()
  const params = useParams()
  const providerID = useMemo(() => Routes.getProviderId(params), [params])
  const [[provider, { error }]] = useProvider(providerID)
  const providerOptionsQuery = useQuery({
    queryKey: [QueryKeys.PROVIDERS, providerID],
    queryFn: async () => (await client.providers.getOptions(providerID!)).unwrap(),
  })

  if (!exists(provider) || !provider.state?.initialized) {
    return <Spinner />
  }

  if (isError(error)) {
    return (
      <>
        <Text>Whoops, something went wrong</Text>
        <ErrorMessageBox error={error} />
      </>
    )
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
        reuseMachine={!!provider.state.singleMachine}
        options={providerOptionsQuery.data ?? {}}
        optionGroups={provider.config?.optionGroups || []}
        onFinish={() => navigate(Routes.PROVIDERS)}
      />
    </Box>
  )
}
