import { Box, Container, VStack } from "@chakra-ui/react"
import { RefObject, useCallback, useEffect, useRef } from "react"
import { TProviderID } from "../../../types"
import { SetupClonedProvider } from "./SetupClonedProvider"
import { ConfigureProviderOptionsForm } from "./ConfigureProviderOptionsForm"
import { SetupProviderSourceForm } from "./SetupProviderSourceForm"
import { TCloneProviderInfo } from "./types"
import { useSetupProvider } from "./useSetupProvider"

type TSetupProviderStepsProps = Readonly<{
  onFinish?: () => void
  isModal?: boolean
  suggestedProvider?: TProviderID
  cloneProviderInfo?: TCloneProviderInfo
  containerRef?: RefObject<HTMLDivElement>
  onProviderIDChanged?: (id: string | null) => void
}>

export function SetupProviderSteps({
  onFinish,
  suggestedProvider,
  cloneProviderInfo,
  onProviderIDChanged,
  containerRef,
  isModal = false,
}: TSetupProviderStepsProps) {
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

  useEffect(() => {
    if (state.providerID) {
      onProviderIDChanged?.(state.providerID)

      return () => onProviderIDChanged?.(null)
    }
  }, [onProviderIDChanged, state.providerID])

  const scrollToElement = useCallback((el: HTMLElement | null) => {
    if (!openLockRef.current) {
      openLockRef.current = true
      setTimeout(
        () =>
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
      {cloneProviderInfo ? (
        <SetupClonedProvider
          cloneProviderInfo={cloneProviderInfo}
          reset={reset}
          onFinish={(result) => {
            completeSetupProvider(result)
            scrollToElement(configureProviderRef.current)
          }}
        />
      ) : (
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
      )}

      <VStack align="start" spacing={8} marginTop={4} width="full">
        <Box width="full" ref={configureProviderRef}>
          {state.currentStep === "configure-provider" && (
            <VStack align="start" width="full">
              <ConfigureProviderOptionsForm
                isModal={isModal}
                addProvider={true}
                providerID={state.providerID}
                isDefault={true}
                reuseMachine={true}
                containerRef={containerRef}
                onFinish={completeConfigureProvider}
              />
            </VStack>
          )}
        </Box>
      </VStack>
    </Container>
  )
}
