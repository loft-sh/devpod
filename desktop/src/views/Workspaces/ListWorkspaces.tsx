import { Button, Text, VStack } from "@chakra-ui/react"
import { useMemo } from "react"
import { useNavigate } from "react-router"
import { useWorkspaces } from "../../contexts"
import { exists } from "../../lib"
import { Routes } from "../../routes"
import { TWorkspace } from "../../types"
import { WorkspaceCard } from "./WorkspaceCard"

type TWorkspacesInfo = Readonly<{
  workspaceCards: TWorkspace[]
}>

export function ListWorkspaces() {
  const navigate = useNavigate()
  const workspaces = useWorkspaces() // TODO: add loading state for workspaces
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

      acc.workspaceCards.push(workspace)

      return acc
    }, empty)
  }, [workspaces])

  return (
    <VStack align="start" marginBottom="12">
      {workspaceCards.length === 0 ? (
        <VStack>
          <Text>No workspaces found. Click here to create one</Text>
          <Button onClick={() => navigate(Routes.WORKSPACE_CREATE)}>Create Workspace</Button>
        </VStack>
      ) : (
        workspaceCards.map((workspace) => (
          <WorkspaceCard key={workspace.id} workspaceID={workspace.id} />
        ))
      )}
    </VStack>
  )
}
