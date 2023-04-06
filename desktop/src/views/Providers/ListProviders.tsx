import { Grid, useToken } from "@chakra-ui/react"
import { useMemo } from "react"
import { useProviders } from "../../contexts"
import { exists } from "../../lib"
import { TProviderID } from "../../types"
import { ProviderCard } from "./ProviderCard"

type TProviderInfo = Readonly<{ name: TProviderID }>
export function ListProviders() {
  const cardSize = useToken("sizes", "56")
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
    <Grid gridGap={4} gridTemplateColumns={`repeat(auto-fill, ${cardSize})`}>
      {providersInfo.map(({ name }) => (
        <ProviderCard key={name} id={name} provider={providers?.[name]} remove={remove} />
      ))}
    </Grid>
  )
}
