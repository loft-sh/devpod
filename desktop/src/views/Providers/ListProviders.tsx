import { SimpleGrid } from "@chakra-ui/react"
import { useMemo } from "react"
import { useProviders } from "../../contexts"
import { exists } from "../../lib"
import { TProviderID } from "../../types"
import { ProviderCard } from "./ProviderCard"

type TProviderInfo = Readonly<{ name: TProviderID }>
export function ListProviders() {
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

  return (
    <SimpleGrid spacing={6} templateColumns="repeat(auto-fill, minmax(20rem, 1fr))">
      {providersInfo.map(({ name }) => (
        <ProviderCard key={name} id={name} provider={providers?.[name]} remove={remove} />
      ))}
    </SimpleGrid>
  )
}
