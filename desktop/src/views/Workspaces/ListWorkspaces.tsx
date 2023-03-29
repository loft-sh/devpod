import { VStack } from "@chakra-ui/react"
import { useMemo } from "react"
import { useWorkspaces } from "../../contexts"
import { exists } from "../../lib"
import { TWorkspace } from "../../types"
import { Workspace } from "./Workspace"

type TWorkspacesInfo = Readonly<{
  workspaceCards: TWorkspace[]
}>

export function ListWorkspaces() {
  const [workspaces] = useWorkspaces()
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
    <>
      <VStack align="start" marginBottom="12">
        {workspaceCards.map((workspace) => (
          <Workspace key={workspace.id} workspace={workspace} />
        ))}
      </VStack>
    </>
  )
}
