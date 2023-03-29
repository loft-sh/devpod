import { Button, Link, ListItem, Text, UnorderedList, VStack } from "@chakra-ui/react"
import { useMemo } from "react"
import { Link as RouterLink } from "react-router-dom"
import { useWorkspaces } from "../../contexts"
import { exists } from "../../lib"
import { Routes } from "../../routes"
import { TWorkspace, TWorkspaceID } from "../../types"

type TWorkspacesInfo = Readonly<{
  workspaceCards: TWorkspaceRow[]
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
  const { workspaceCards } = useMemo<TWorkspacesInfo>(() => {
    const empty: TWorkspacesInfo = { workspaceCards: [] }
    if (!exists(workspaces)) {
      return empty
    }

    return workspaces.reduce<TWorkspacesInfo>((acc, workspace) => {
      const { id } = workspace
      if (!exists(id)) {
        return acc
      }

      const row = WorkspaceRow.fromWorkspace(workspace)
      acc.workspaceCards.push(row)

      return acc
    }, empty)
  }, [workspaces])

  return (
    <>
      <VStack align="start" marginBottom="12">
        <UnorderedList listStyleType="none">
          {workspaceCards.map(({ id, name, ide, providerName, status }) => (
            <ListItem key={name}>
              <Link as={RouterLink} to={Routes.toWorkspace(name)}>
                <Text fontWeight="bold">{name}</Text>
              </Link>

              {exists(providerName) && <Text>Provider: {providerName}</Text>}
              <Text>Status: {status}</Text>

              <Button
                onClick={() => start.run({ workspaceID: id, config: { ideConfig: ide } })}
                isLoading={start.status === "loading"}>
                Start
              </Button>
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
