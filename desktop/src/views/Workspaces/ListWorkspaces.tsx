import { Button, HStack, VStack } from "@chakra-ui/react"
import { useCallback, useMemo, useState } from "react"
import { client } from "../../client"
import { useWorkspaces } from "../../contexts"
import { exists } from "../../lib"
import { TWorkspace } from "../../types"
import { WorkspaceCard } from "./WorkspaceCard"

type TWorkspacesInfo = Readonly<{
  workspaceCards: TWorkspace[]
}>

export function ListWorkspaces() {
  const [selectedWorkspaces, setSelectedWorkspaces] = useState<readonly TWorkspace[]>([])
  const workspaces = useWorkspaces()
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

  const handleWorkspaceSelectionChanged = useCallback((workspace: TWorkspace) => {
    return (isSelected: boolean) => {
      setSelectedWorkspaces((prev) => {
        if (!isSelected) {
          return prev.filter((w) => w.id !== workspace.id)
        }

        return [...prev, workspace]
      })
    }
  }, [])

  return (
    <>
      <VStack align="start" marginBottom="12">
        {import.meta.env.DEV && (
          <BatchDeleteWorkspaces
            allWorkspaces={workspaces}
            selectedWorkspaces={selectedWorkspaces}
          />
        )}

        {workspaceCards.map((workspace) => (
          <WorkspaceCard
            key={workspace.id}
            workspaceID={workspace.id}
            onSelectionChange={
              import.meta.env.DEV ? handleWorkspaceSelectionChanged(workspace) : undefined
            }
          />
        ))}
      </VStack>
    </>
  )
}

type TBatchDeleteWorkspaces = Readonly<{
  allWorkspaces: readonly TWorkspace[]
  selectedWorkspaces: readonly TWorkspace[]
}>
function BatchDeleteWorkspaces({ allWorkspaces, selectedWorkspaces }: TBatchDeleteWorkspaces) {
  const handleDeleteSelectedClicked = useCallback(() => {
    client.workspaces.removeMany(selectedWorkspaces)
  }, [selectedWorkspaces])
  const handleDeleteAllClicked = useCallback(() => {
    client.workspaces.removeMany(allWorkspaces)
  }, [allWorkspaces])

  return (
    <HStack width="full" justifyContent="end" minHeight="8" marginBottom="12">
      {selectedWorkspaces.length > 0 && (
        <Button colorScheme="red" onClick={handleDeleteSelectedClicked}>
          Delete {selectedWorkspaces.length}
        </Button>
      )}
      {allWorkspaces.length > 0 && (
        <Button colorScheme="red" onClick={handleDeleteAllClicked}>
          Delete All
        </Button>
      )}
    </HStack>
  )
}
