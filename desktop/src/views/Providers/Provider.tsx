import { Box, Spinner } from "@chakra-ui/react"
import { useEffect, useMemo, useState } from "react"
import { useNavigate, useParams } from "react-router"
import { client } from "../../client"
import { useProvider } from "../../contexts"
import { exists } from "../../lib"
import { Routes } from "../../routes"
import { ConfigureProviderOptionsForm } from "./AddProvider/ConfigureProviderOptionsForm"
import { TProviderOptions } from "../../types"

export function Provider() {
  const navigate = useNavigate()
  const params = useParams()
  const providerID = useMemo(() => Routes.getProviderId(params), [params])
  const [provider] = useProvider(providerID)
  const [providerOptions, setProviderOptions] = useState<TProviderOptions | undefined>()
  useEffect(() => {
    ;(async () => {
      const result = await client.providers.getOptions(providerID!)
      if (result.err) {
        return
      }

      setProviderOptions(result.val!)
    })()
  }, [providerID])

  if (!exists(provider) || !providerOptions) {
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
        options={providerOptions ?? {}}
        optionGroups={provider.config?.optionGroups || []}
        onFinish={() => navigate(Routes.PROVIDERS)}
      />
    </Box>
  )
}
