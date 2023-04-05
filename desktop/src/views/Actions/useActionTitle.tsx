import { IconButton } from "@chakra-ui/react"
import { useMemo } from "react"
import { useMatch, useNavigate } from "react-router"
import { TViewTitle } from "../../components"
import { ArrowLeft } from "../../icons"
import { exists } from "../../lib"
import { Routes } from "../../routes"

export function useActionTitle(): TViewTitle | null {
  const navigate = useNavigate()

  const matchAction = useMatch(Routes.ACTION)

  return useMemo<TViewTitle | null>(() => {
    if (!exists(matchAction)) {
      return null
    }

    return {
      label: "TODO: get action title",
      priority: "regular",
      leadingAction: (
        <IconButton
          variant="ghost"
          aria-label="Navigate back to Workspaces"
          icon={<ArrowLeft />}
          onClick={() => {}}
          // TODO: navigate to wherever the user came from or to workspace root
        />
      ),
    }
  }, [matchAction])
}
