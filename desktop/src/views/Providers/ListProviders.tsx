import { Button, Text, VStack, Wrap, WrapItem } from "@chakra-ui/react"
import { useMemo } from "react"
import { useProviders } from "../../contexts"
import { canHealthCheck, exists } from "../../lib"
import { TProvider, TProviderID } from "../../types"
import { useSetupProviderModal } from "../Providers/useSetupProviderModal"
import { ProviderCard } from "./ProviderCard"

type TProviderInfo = Readonly<{ id: TProviderID; data: TProvider }>
export function ListProviders() {
  const [[providers], { remove }] = useProviders()
  const { show: showSetupProvider, modal } = useSetupProviderModal()
  const providersInfo = useMemo<readonly TProviderInfo[]>(() => {
    if (!exists(providers)) {
      return []
    }

    return Object.entries(providers)
      .filter(([, details]) => details.state?.initialized && !canHealthCheck(details.config))
      .map(([id, data]) => {
        return { id, data }
      })
  }, [providers])

  return (
    <>
      {providersInfo.length === 0 ? (
        <VStack>
          <Text>No providers found. Click here to add one</Text>
          <Button onClick={() => showSetupProvider({ isStrict: false })}>Add Provider</Button>
        </VStack>
      ) : (
        <Wrap paddingBottom={8}>
          {providersInfo.map(({ id, data }) => (
            <WrapItem key={id}>
              <ProviderCard id={id} provider={data} remove={remove} />
            </WrapItem>
          ))}
        </Wrap>
      )}

      {modal}
    </>
  )
}
