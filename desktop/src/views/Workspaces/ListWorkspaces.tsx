import { Button, Heading, Link, ListItem, Text, UnorderedList, VStack } from "@chakra-ui/react"
import { useMemo } from "react"
import { Link as RouterLink } from "react-router-dom"
import { useWorkspaces } from "../../contexts/DevPodContext/DevPodContext"
import { exists } from "../../lib"
import { Routes } from "../../routes"
import { TWorkspace, TWorkspaceID } from "../../types"

type TWorkspacesViewModel = Readonly<{
  activeProviderCards: TWorkspaceRow[]
  inactiveProviderRows: TWorkspaceRow[]
}>
type TWorkspaceRow = Readonly<{
  id: TWorkspaceID
  name: string
  providerName: string | null
  status: string
  ide: TWorkspace["ide"]
}>

export function ListWorkspaces() {
  const [[workspaces], { start, stop, remove }] = useWorkspaces()
  const { activeProviderCards, inactiveProviderRows } = useMemo<TWorkspacesViewModel>(() => {
    const empty: TWorkspacesViewModel = { inactiveProviderRows: [], activeProviderCards: [] }
    if (!exists(workspaces)) {
      return empty
    }

    return workspaces.reduce<TWorkspacesViewModel>((acc, workspace) => {
      const { id, status } = workspace
      if (!exists(id)) {
        return acc
      }

      const row = WorkspaceRow.fromWorkspace(workspace)

      if (status === "Running") {
        acc.activeProviderCards.push(row)

        return acc
      }

      acc.inactiveProviderRows.push(row)

      return acc
    }, empty)
  }, [workspaces])

  return (
    <>
      <VStack align="start" marginBottom="12">
        <Heading as="h3" size="md" marginBottom="4">
          Active
        </Heading>
        <UnorderedList listStyleType="none">
          {activeProviderCards.map(({ id, name, providerName, status }) => (
            <ListItem key={name}>
              <Link as={RouterLink} to={Routes.toWorkspace(name)}>
                <Text fontWeight="bold">{name}</Text>
              </Link>

              {exists(providerName) && <Text>Provider: {providerName}</Text>}
              <Text>Status: {status}</Text>

              <Button
                onClick={() => stop.run({ workspaceID: id })}
                isLoading={stop.status === "loading"}>
                Stop
              </Button>
              <Button
                colorScheme="red"
                onClick={() => remove.run({ workspaceID: id })}
                isLoading={remove.status === "loading"}>
                Delete
              </Button>
            </ListItem>
          ))}
        </UnorderedList>
      </VStack>

      <VStack align="start" marginBottom="4">
        <Heading as="h3" size="md" marginBottom="4">
          Inactive
        </Heading>

        <UnorderedList listStyleType="none">
          {inactiveProviderRows.map(({ id, name, providerName, status, ide }) => (
            <ListItem key={name}>
              <Link as={RouterLink} to={Routes.toWorkspace(name)}>
                <Text fontWeight="bold">{name}</Text>
              </Link>

              {exists(providerName) && <Text>Provider: {providerName}</Text>}
              <Text>Status: {status}</Text>

              <Button
                onClick={() =>
                  start.run({ workspaceID: id, config: { ideConfig: ide }, onStream: () => {} })
                }
                isLoading={start.status === "loading"}>
                Start
              </Button>
              <Button
                colorScheme="red"
                onClick={() => remove.run({ workspaceID: id })}
                isLoading={remove.status === "loading"}>
                Delete
              </Button>
            </ListItem>
          ))}
        </UnorderedList>
      </VStack>
    </>
  )
}

const WorkspaceRow = {
  fromWorkspace(workspace: TWorkspace): TWorkspaceRow {
    return {
      id: workspace.id,
      name: workspace.id,
      providerName: workspace.provider?.name ?? null,
      status: workspace.status ?? "unknown",
      ide: workspace.ide,
    }
  },
}
