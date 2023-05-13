import { Box, Container, VStack } from "@chakra-ui/react"
import { useCallback, useEffect, useRef } from "react"
import { TProviderID } from "../../../types"
import { ConfigureProviderOptionsForm } from "./ConfigureProviderOptionsForm"
import { SetupProviderSourceForm } from "./SetupProviderSourceForm"
import { useSetupProvider } from "./useSetupProvider"

export function SetupProviderSteps({
  onFinish,
  suggestedProvider,
  isModal = false,
}: Readonly<{ onFinish?: () => void; suggestedProvider?: TProviderID; isModal?: boolean }>) {
  const openLockRef = useRef(false)
  const configureProviderRef = useRef<HTMLDivElement>(null)
  const {
    state,
    reset,
    completeSetupProvider,
    completeConfigureProvider,
    removeDanglingProviders,
  } = useSetupProvider()

  useEffect(() => {
    if (state.currentStep === "done") {
      onFinish?.()
    }
  }, [onFinish, state.currentStep])

  const scrollToElement = useCallback((el: HTMLElement | null) => {
    if (!openLockRef.current) {
      openLockRef.current = true
      setTimeout(() =>
        el?.scrollIntoView({
          behavior: "smooth",
          block: "start",
          inline: "nearest",
        })
      )
    }
  }, [])

  return (
    <Container maxWidth="container.lg">
      <VStack align="start" spacing={8} width="full">
        <SetupProviderSourceForm
          suggestedProvider={suggestedProvider}
          reset={reset}
          onFinish={(result) => {
            completeSetupProvider(result)
            scrollToElement(configureProviderRef.current)
          }}
          removeDanglingProviders={removeDanglingProviders}
        />
      </VStack>

      <VStack align="start" spacing={8} marginTop={6} width="full">
        <Box width="full" ref={configureProviderRef}>
          {state.currentStep === "configure-provider" && (
            <VStack align="start" width="full">
              <ConfigureProviderOptionsForm
                isModal={isModal}
                addProvider={true}
                providerID={state.providerID}
                isDefault={true}
                reuseMachine={true}
                options={state.options}
                optionGroups={state.optionGroups}
                onFinish={completeConfigureProvider}
              />
            </VStack>
          )}
        </Box>
      </VStack>
    </Container>
  )
}
