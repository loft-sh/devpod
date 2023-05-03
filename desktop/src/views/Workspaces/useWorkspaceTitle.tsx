import { Button, IconButton } from "@chakra-ui/react"
import { useCallback, useMemo } from "react"
import { useMatch, useNavigate } from "react-router"
import { TViewTitle } from "../../components"
import { ArrowLeft, Plus } from "../../icons"
import { exists } from "../../lib"
import { Routes } from "../../routes"

export function useWorkspaceTitle(): TViewTitle | null {
  const navigate = useNavigate()

  const matchWorkspacesRoot = useMatch(Routes.WORKSPACES)
  const matchCreateWorkspace = useMatch(Routes.WORKSPACE_CREATE)

  const navigateToWorkspacesRoot = useCallback(() => {
    navigate(Routes.WORKSPACES)
  }, [navigate])

  const navigateBackAction = useMemo(() => {
    return (
      <IconButton
        variant="ghost"
        aria-label="Navigate back to Workspaces"
        icon={<ArrowLeft />}
        onClick={navigateToWorkspacesRoot}
      />
    )
  }, [navigateToWorkspacesRoot])

  return useMemo<TViewTitle | null>(() => {
    if (exists(matchWorkspacesRoot)) {
      return {
        label: "Workspaces",
        priority: "high",
        trailingAction: (
          <Button
            size="sm"
            variant="outline"
            aria-label="Create new workspace"
            leftIcon={<Plus boxSize={5} />}
            onClick={() => navigate(Routes.WORKSPACE_CREATE)}>
            Create
          </Button>
        ),
      }
    }

    if (exists(matchCreateWorkspace)) {
      return {
        label: "Create Workspace",
        priority: "regular",
        leadingAction: navigateBackAction,
      }
    }

    return null
  }, [matchCreateWorkspace, matchWorkspacesRoot, navigate, navigateBackAction])
}
