import { Box, Heading, VStack } from "@chakra-ui/react"
import { useEffect, useRef } from "react"
import { useNavigate } from "react-router-dom"
import { CollapsibleSection } from "../../../components"
import { Routes } from "../../../routes"
import { ConfigureProviderOptionsForm } from "./ConfigureProviderOptionsForm"
import { SetupProviderSourceForm } from "./SetupProviderSourceForm"
import { useSetupProvider } from "./useSetupProvider"

export function AddProvider() {
  const navigate = useNavigate()
  const openLockRef = useRef(false)
  const { state, completeFirstStep, completeSecondStep } = useSetupProvider()

  useEffect(() => {
    if (state.currentStep === "done") {
      navigate(Routes.toProvider(state.providerID))
    }
  }, [navigate, state.currentStep, state.providerID])

  return (
    <Box paddingBottom={80}>
      <VStack align="start" spacing={8} width="full">
        <CollapsibleSection isOpen title={<Heading size="md">1. Setup Provider Source</Heading>}>
          <SetupProviderSourceForm onFinish={completeFirstStep} />
        </CollapsibleSection>
      </VStack>

      <VStack align="start" spacing={8} marginTop={6} width="full">
        <CollapsibleSection
          isDisabled={state.currentStep === 1}
          isOpen={state.currentStep === 2}
          onOpenChange={(isOpen, el) => {
            if (isOpen && !openLockRef.current) {
              openLockRef.current = true
              setTimeout(() =>
                el?.scrollIntoView({
                  behavior: "smooth",
                  block: "start",
                  inline: "nearest",
                })
              )
            }
          }}
          title={<Heading size="md">2. Configure Provider</Heading>}>
          <VStack align="start" width="full">
            {state.currentStep === 2 && (
              <ConfigureProviderOptionsForm
                initializeProvider
                providerID={state.providerID}
                options={state.options}
                optionGroups={state.optionGroups}
                onFinish={completeSecondStep}
              />
            )}
          </VStack>
        </CollapsibleSection>
      </VStack>
    </Box>
  )
}
