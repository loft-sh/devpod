import {
  Accordion,
  AccordionButton,
  AccordionIcon,
  AccordionItem,
  AccordionPanel,
  Box,
  Code,
  ListItem,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Text,
  UnorderedList,
  useTheme,
  useToken,
  VStack,
} from "@chakra-ui/react"
import { ReactNode, useMemo } from "react"
import { useProviders, useWorkspaces } from "./DevpodContext"
import { exists } from "./helpers"
import { DevpodIcon } from "./icons"

export function App() {
  return (
    <Layout>
      <Tabs>
        <TabList>
          <Tab>Workspaces</Tab>
          <Tab>Providers</Tab>
        </TabList>

        <TabPanels>
          <TabPanel>
            <WorkspacesTab />
          </TabPanel>
          <TabPanel>
            <ProvidersTab />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </Layout>
  )
}

function Layout({ children }: Readonly<{ children?: ReactNode }>) {
  return (
    <VStack spacing={4} height="100vh">
      <Header />
      <Box width="full" height="full" overflowY="auto">
        {children}
      </Box>
    </VStack>
  )
}

function Header() {
  // FIXME: refactor into type-safe hook
  const iconColor = useToken("colors", "primary")

  return (
    <Box width="full" paddingX={4} paddingY={4}>
      <DevpodIcon boxSize={8} color={iconColor} />
    </Box>
  )
}

type TProviderRow = Readonly<{ name: string; options: string }>
function ProvidersTab() {
  const providers = useProviders()
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
  )
}

type TWorkspaceRow = Readonly<{ name: string; providerName: string | null }>
function WorkspacesTab() {
  const workspaces = useWorkspaces()
  const providerRows = useMemo<readonly TWorkspaceRow[]>(() => {
    if (!exists(workspaces)) {
      return []
    }

    return workspaces.reduce<readonly TWorkspaceRow[]>((acc, { id, provider }) => {
      if (!exists(id)) {
        return acc
      }

      return [...acc, { providerName: provider?.name ?? null, name: id }]
    }, [])
  }, [workspaces])

  return (
    <UnorderedList>
      {providerRows.map((row) => (
        <ListItem key={row.name}>
          <Text fontWeight="bold">{row.name}</Text>

          {exists(row.providerName) && <Text>Provider: {row.providerName}</Text>}
        </ListItem>
      ))}
    </UnorderedList>
  )
}
