import { Box, Heading, VStack } from "@chakra-ui/react"
import { useEffect, useRef } from "react"
import { CollapsibleSection } from "../../../components"
import { ConfigureProviderOptionsForm } from "./ConfigureProviderOptionsForm"
import { SetupProviderSourceForm } from "./SetupProviderSourceForm"
import { useAddProvider } from "./useAddProvider"

export function AddProvider() {
  const openLockRef = useRef(false)
  const { state, completeFirstStep, completeSecondStep } = useAddProvider()

  useEffect(() => {
    if (state.currentStep === "done") {
      navigate()
    }
  })

  return (
    <Box paddingBottom={80}>
      <VStack align="start" spacing={8} width="full">
        <CollapsibleSection isOpen title={<Heading size="md">1. Setup Provider Source</Heading>}>
          <SetupProviderSourceForm onFinish={completeFirstStep} />
        </CollapsibleSection>
      </VStack>

      <VStack align="start" spacing={8} marginTop={6} width="full">
        <CollapsibleSection
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
                providerID={state.providerID}
                options={state.options}
                onFinish={completeSecondStep}
              />
            )}
          </VStack>
        </CollapsibleSection>
      </VStack>
    </Box>
  )
}
