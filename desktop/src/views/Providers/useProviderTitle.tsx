import { Button, IconButton } from "@chakra-ui/react"
import { useCallback, useMemo } from "react"
import { useMatch, useNavigate } from "react-router"
import { TViewTitle } from "../../components"
import { ArrowLeft, Plus } from "../../icons"
import { exists } from "../../lib"
import { Routes } from "../../routes"
import { useSetupProviderModal } from "./useSetupProviderModal"

export function useProviderTitle(): TViewTitle | null {
  const navigate = useNavigate()

  const matchProviderRoot = useMatch(Routes.PROVIDERS)
  const matchProvider = useMatch(Routes.PROVIDER)
  const { modal, show: showSetupProvider } = useSetupProviderModal()

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
          <>
            <Button
              size="sm"
              variant="outline"
              aria-label="Add provider"
              leftIcon={<Plus />}
              onClick={() => showSetupProvider({ isStrict: false })}>
              Add
            </Button>
            {modal}
          </>
        ),
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
  }, [matchProvider, matchProviderRoot, modal, navigateBackAction, showSetupProvider])
}
