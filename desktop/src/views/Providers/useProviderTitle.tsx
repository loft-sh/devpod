import { Button, IconButton } from "@chakra-ui/react"
import { useCallback, useMemo } from "react"
import { useMatch, useNavigate } from "react-router"
import { TViewTitle } from "../../components"
import { ArrowLeft, Plus } from "../../icons"
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
        icon={<ArrowLeft />}
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
          <Button
            size="sm"
            variant="outline"
            aria-label="Add provider"
            leftIcon={<Plus />}
            onClick={() => navigate(Routes.PROVIDER_ADD)}>
            Add
          </Button>
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
