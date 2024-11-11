import { IconButton } from "@chakra-ui/react"
import { useMemo } from "react"
import { useLocation, useMatch, useNavigate } from "react-router-dom"
import { TViewTitle } from "../../components"
import { getAction, useWorkspaceStore } from "../../contexts"
import { ArrowLeft } from "../../icons"
import { exists, getActionDisplayName } from "../../lib"
import { Routes } from "../../routes"

export function useActionTitle(): TViewTitle | null {
  const { store } = useWorkspaceStore()
  const navigate = useNavigate()
  const location = useLocation()

  const matchAction = useMatch(Routes.ACTION)

  return useMemo<TViewTitle | null>(() => {
    if (!exists(matchAction)) {
      return null
    }
    const maybeActionID = Routes.getActionID(matchAction.params)
    if (!maybeActionID) {
      return null
    }
    const maybeAction = getAction(maybeActionID, store)
    if (maybeAction === undefined) {
      return null
    }

    const targetRoute =
      // Unfortunately `Location` isn't typed, so be careful if you change this
      exists(location.state?.origin) && location.state?.origin !== ""
        ? location.state?.origin
        : Routes.WORKSPACES

    return {
      label: getActionDisplayName(maybeAction),
      priority: "regular",
      leadingAction: (
        <IconButton
          variant="ghost"
          aria-label="Navigate back to Workspaces"
          icon={<ArrowLeft />}
          onClick={() => {
            navigate(targetRoute)
          }}
        />
      ),
    }
  }, [location.state?.origin, matchAction, navigate, store])
}
