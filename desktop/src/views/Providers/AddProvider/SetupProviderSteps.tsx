import { Heading, VStack } from "@chakra-ui/react"
import { useEffect, useRef } from "react"
import { CollapsibleSection } from "../../../components"
import { ConfigureProviderOptionsForm } from "./ConfigureProviderOptionsForm"
import { SetupProviderSourceForm } from "./SetupProviderSourceForm"
import { useSetupProvider } from "./useSetupProvider"
import { TProviderID } from "../../../types"

export function SetupProviderSteps({
  onFinish,
  suggestedProvider,
}: Readonly<{ onFinish?: () => void; suggestedProvider?: TProviderID }>) {
  const openLockRef = useRef(false)
  const { state, reset, completeFirstStep, completeSecondStep } = useSetupProvider()

  useEffect(() => {
    if (state.currentStep === "done") {
      onFinish?.()
    }
  }, [onFinish, state.currentStep])

  return (
    <>
      <VStack align="start" spacing={8} width="full">
        <Heading size="md">1. Setup Provider Source</Heading>
        <SetupProviderSourceForm
          state={state}
          suggestedProvider={suggestedProvider}
          reset={reset}
          onFinish={completeFirstStep}
        />
      </VStack>

      <VStack align="start" spacing={8} marginTop={6} width="full">
        <CollapsibleSection
          headerProps={{ pointerEvents: "none", padding: "0" }}
          contentProps={{ paddingLeft: "0" }}
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
                addProvider={true}
                providerID={state.providerID}
                isDefault={true}
                reuseMachine={true}
                options={state.options}
                optionGroups={state.optionGroups}
                onFinish={completeSecondStep}
              />
            )}
          </VStack>
        </CollapsibleSection>
      </VStack>
    </>
  )
}
