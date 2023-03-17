import { AddIcon, ArrowBackIcon } from "@chakra-ui/icons"
import { IconButton } from "@chakra-ui/react"
import { ReactElement, useCallback, useMemo } from "react"
import { useMatch, useNavigate } from "react-router"
import { exists } from "../../lib"
import { Routes } from "../../routes"

type TTitle = Readonly<{
  label: string
  priority: "high" | "regular"
  leadingAction?: ReactElement
  trailingAction?: ReactElement
}>

export function useWorkspaceTitle(): TTitle | null {
  const navigate = useNavigate()

  const matchWorkspacesRoot = useMatch(Routes.WORKSPACES)
  const matchCreateWorkspace = useMatch(Routes.WORKSPACE_CREATE)
  const matchWorkspace = useMatch(Routes.WORKSPACE)

  const navigateToWorkspacesRoot = useCallback(() => {
    navigate(Routes.WORKSPACES)
  }, [navigate])

  const navigateBackAction = useMemo(() => {
    return (
      <IconButton
        variant="ghost"
        aria-label="Navigate back to Workspaces"
        icon={<ArrowBackIcon boxSize="6" />}
        onClick={navigateToWorkspacesRoot}
      />
    )
  }, [navigateToWorkspacesRoot])

  return useMemo<TTitle | null>(() => {
    if (exists(matchWorkspacesRoot)) {
      return {
        label: "Workspaces",
        priority: "high",
        trailingAction: (
          <IconButton
            aria-label="Create Workspace"
            icon={<AddIcon />}
            onClick={() => navigate(Routes.WORKSPACE_CREATE)}
          />
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

    if (exists(matchWorkspace)) {
      const maybeWorkspaceId = Routes.getWorkspaceId(matchWorkspace.params)

      return {
        label: maybeWorkspaceId ?? "Unknown Workspace",
        priority: "regular",
        leadingAction: navigateBackAction,
      }
    }

    return null
  }, [matchCreateWorkspace, matchWorkspace, matchWorkspacesRoot, navigate, navigateBackAction])
}
