import { AddIcon, ArrowBackIcon } from "@chakra-ui/icons"
import { IconButton } from "@chakra-ui/react"
import { useCallback, useMemo } from "react"
import { useMatch, useNavigate } from "react-router"
import { TViewTitle } from "../../components"
import { exists } from "../../lib"
import { Routes } from "../../routes"

export function useProviderTitle(): TViewTitle | null {
  const navigate = useNavigate()

  const matchProviderRoot = useMatch(Routes.PROVIDERS)
  const matchAddProvider = useMatch(Routes.PROVIDER_ADD)
  const matchProvider = useMatch(Routes.PROVIDER)

  const navigateToProviderRoot = useCallback(() => {
    navigate(Routes.PROVIDERS)
  }, [navigate])

  const navigateBackAction = useMemo(() => {
    return (
      <IconButton
        variant="ghost"
        aria-label="Navigate back to providers"
        icon={<ArrowBackIcon boxSize="6" />}
        onClick={navigateToProviderRoot}
      />
    )
  }, [navigateToProviderRoot])

  return useMemo<TViewTitle | null>(() => {
    if (exists(matchProviderRoot)) {
      return {
        label: "Providers",
        priority: "high",
        trailingAction: (
          <IconButton
            aria-label="Create Provider"
            icon={<AddIcon />}
            onClick={() => navigate(Routes.PROVIDER_ADD)}
          />
        ),
      }
    }

    if (exists(matchAddProvider)) {
      return {
        label: "Add Provider",
        priority: "regular",
        leadingAction: navigateBackAction,
      }
    }

    if (exists(matchProvider)) {
      const maybeWorkspaceId = Routes.getProviderId(matchProvider.params)

      return {
        label: maybeWorkspaceId ?? "Unknown Provider",
        priority: "regular",
        leadingAction: navigateBackAction,
      }
    }

    return null
  }, [matchAddProvider, matchProvider, matchProviderRoot, navigate, navigateBackAction])
}
