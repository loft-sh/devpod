import {
  Accordion,
  AccordionButton,
  AccordionIcon,
  AccordionItem,
  AccordionPanel,
  Code,
  Text,
} from "@chakra-ui/react"
import { useMemo } from "react"
import { useProviders } from "../../contexts"
import { exists } from "../../lib"

type TProviderRow = Readonly<{ name: string; options: string }>
export function Providers() {
  const [providers] = useProviders()
  const providerRows = useMemo<readonly TProviderRow[]>(() => {
    const maybeProviders = providers?.providers
    if (!exists(maybeProviders)) {
      return []
    }

    return Object.entries(maybeProviders).map(([name, details]) => {
      return { name, options: JSON.stringify(details, null, 2) }
    })
  }, [providers])

  return (
    <>
      <div>Providers</div>
      <Accordion allowMultiple>
        {providerRows.map((row) => (
          <AccordionItem key={row.name}>
            <AccordionButton>
              <AccordionIcon />
              <Text>{row.name}</Text>
            </AccordionButton>
            <AccordionPanel>
              <Code padding={4} whiteSpace="pre" display="block" borderRadius="md">
                {row.options}
              </Code>
            </AccordionPanel>
          </AccordionItem>
        ))}
      </Accordion>
    </>
  )
}
