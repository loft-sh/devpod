import { IconButton } from "@chakra-ui/react"
import { useMemo } from "react"
import { useLocation, useMatch, useNavigate } from "react-router-dom"
import { TViewTitle } from "../../components"
import { getAction } from "../../contexts"
import { ArrowLeft } from "../../icons"
import { exists, getActionDisplayName } from "../../lib"
import { Routes } from "../../routes"

export function useActionTitle(): TViewTitle | null {
  const navigate = useNavigate()
  const location = useLocation()

  console.log(location)

  const matchAction = useMatch(Routes.ACTION)

  return useMemo<TViewTitle | null>(() => {
    if (!exists(matchAction)) {
      return null
    }
    const maybeActionID = Routes.getActionID(matchAction.params)
    if (!maybeActionID) {
      return null
    }
    const maybeAction = getAction(maybeActionID)
    if (maybeAction === undefined) {
      return null
    }

    return {
      label: getActionDisplayName(maybeAction),
      priority: "regular",
      leadingAction: (
        <IconButton
          variant="ghost"
          aria-label="Navigate back to Workspaces"
          icon={<ArrowLeft />}
          onClick={() => {
            if (location.key !== "default") {
              navigate(-1)
            } else {
              navigate(Routes.WORKSPACES)
            }
          }}
        />
      ),
    }
  }, [location.key, matchAction, navigate])
}
