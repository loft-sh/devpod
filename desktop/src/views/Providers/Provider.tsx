import { Container, Spinner } from "@chakra-ui/react"
import { useMemo, useRef } from "react"
import { useNavigate, useParams } from "react-router"
import { useProvider } from "../../contexts"
import { exists } from "../../lib"
import { Routes } from "../../routes"
import { ConfigureProviderOptionsForm } from "./AddProvider"

export function Provider() {
  const navigate = useNavigate()
  const params = useParams()
  const providerID = useMemo(() => Routes.getProviderId(params), [params])
  const [provider] = useProvider(providerID)
  const containerRef = useRef<HTMLDivElement>(null)
  if (!exists(provider)) {
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
        onFinish={() => navigate(Routes.PROVIDERS)}
      />
    </Container>
  )
}
