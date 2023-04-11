import { Button, Text, VStack, Wrap, WrapItem } from "@chakra-ui/react"
import { useMemo } from "react"
import { useProviders } from "../../contexts"
import { exists } from "../../lib"
import { TProviderID } from "../../types"
import { ProviderCard } from "./ProviderCard"
import { Routes } from "../../routes"
import { useNavigate } from "react-router"

type TProviderInfo = Readonly<{ name: TProviderID }>
export function ListProviders() {
  const navigate = useNavigate()
  const [[providers], { remove }] = useProviders()
  const providersInfo = useMemo<readonly TProviderInfo[]>(() => {
    if (!exists(providers)) {
      return []
    }

    return Object.entries(providers)
      .filter(([, details]) => exists(details.state))
      .map(([name, details]) => {
        return { name, options: JSON.stringify(details.config, null, 2) }
      })
  }, [providers])

  return providersInfo.length === 0 ? (
    <VStack>
      <Text>No providers found. Click here to add one</Text>
      <Button onClick={() => navigate(Routes.PROVIDER_ADD)}>Add Provider</Button>
    </VStack>
  ) : (
    <Wrap>
      {providersInfo.map(({ name }) => (
        <WrapItem key={name}>
          <ProviderCard key={name} id={name} provider={providers?.[name]} remove={remove} />
        </WrapItem>
      ))}
    </Wrap>
  )
}
