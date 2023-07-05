import { Container, Spinner } from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import { useMemo, useRef } from "react"
import { useNavigate, useParams } from "react-router"
import { client } from "../../client"
import { useProvider } from "../../contexts"
import { exists } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import { Routes } from "../../routes"
import { ConfigureProviderOptionsForm } from "./AddProvider"

export function Provider() {
  const navigate = useNavigate()
  const params = useParams()
  const providerID = useMemo(() => Routes.getProviderId(params), [params])
  const [provider] = useProvider(providerID)
  const { data: providerOptions } = useQuery({
    queryKey: QueryKeys.providerOptions(providerID!),
    queryFn: async () => (await client.providers.getOptions(providerID!)).unwrap(),
    enabled: providerID !== undefined,
  })
  const containerRef = useRef<HTMLDivElement>(null)

  if (!exists(provider) || !providerOptions) {
    return <Spinner />
  }

  if (!exists(providerID)) {
    return null
  }

  return (
    <Container width="full" maxWidth="container.lg" ref={containerRef}>
      <ConfigureProviderOptionsForm
        containerRef={containerRef}
        providerID={providerID}
        isDefault={!!provider.default}
        addProvider={false}
        reuseMachine={!!provider.state?.singleMachine}
        options={providerOptions}
        optionGroups={provider.config?.optionGroups || []}
        onFinish={() => navigate(Routes.PROVIDERS)}
      />
    </Container>
  )
}
