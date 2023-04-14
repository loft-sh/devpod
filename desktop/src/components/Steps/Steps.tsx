import { Box, Button, HStack, useColorModeValue } from "@chakra-ui/react"
import {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useEffect,
  useId,
  useMemo,
  useState,
} from "react"

type TStepContext = Readonly<{
  addStep: (id: string, step: ReactNode) => void
}>
type TStep = Readonly<{
  id: string
  node: ReactNode
}>

const StepsContext = createContext<TStepContext>(null!)
type TStepsProps = Readonly<{ onFinish?: VoidFunction; finishText?: string; children: ReactNode }>
export function Steps({ onFinish, finishText, children }: TStepsProps) {
  const [steps, setSteps] = useState<readonly TStep[]>([])
  // Steps are 0 based for easier array indexing
  const [currentStep, setCurrentStep] = useState(0)
  const stepIndicatorInactiveColor = useColorModeValue("gray.200", "gray.700")
  const stepIndicatorActiveColor = useColorModeValue("primary.600", "primary.400")

  const isLastStep = useMemo(() => currentStep >= steps.length - 1, [currentStep, steps.length])
  const isFirstStep = useMemo(() => currentStep <= 0, [currentStep])

  const addStep = useCallback<TStepContext["addStep"]>((id, node) => {
    setSteps((steps) => {
      const newSteps = steps.slice()
      const index = newSteps.findIndex((a) => a.id === id)
      if (index !== -1) {
        newSteps.splice(index, 1, { id, node: node })
      } else {
        newSteps.push({ id, node: node })
      }

      return newSteps
    })

    return () => {
      setSteps((steps) => steps.filter((a) => a.id !== id))
    }
  }, [])

  const handleBackClicked = useCallback(() => {
    if (isFirstStep) {
      return
    }

    setCurrentStep((currentStep) => currentStep - 1)
  }, [isFirstStep])

  const handleNextClicked = useCallback(() => {
    if (isLastStep) {
      onFinish?.()

      return
    }

    setCurrentStep((currentStep) => currentStep + 1)
  }, [isLastStep, onFinish])

  const value = useMemo(() => ({ addStep }), [addStep])

  return (
    <StepsContext.Provider value={value}>
      <>
        {children}

        {steps[currentStep]?.node}

        <HStack width="full" justifyContent="space-between">
          <Button
            variant="ghost"
            isDisabled={isFirstStep}
            visibility={isFirstStep ? "hidden" : "visible"}
            onClick={handleBackClicked}>
            Back
          </Button>
          <HStack width="full" justifyContent="center" gap="2">
            {steps.map((_, i) => (
              <Box
                key={i}
                width="3"
                height="3"
                borderRadius="full"
                marginInlineStart="0 !important"
                backgroundColor={
                  i === currentStep ? stepIndicatorActiveColor : stepIndicatorInactiveColor
                }
              />
            ))}
          </HStack>
          <Button variant={isLastStep ? "primary" : "solid"} onClick={handleNextClicked}>
            {isLastStep ? (finishText !== undefined ? finishText : "Done") : "Next"}
          </Button>
        </HStack>
      </>
    </StepsContext.Provider>
  )
}

type TStepProps = Readonly<{ children: ReactNode }>
export function Step({ children }: TStepProps) {
  const { addStep } = useContext(StepsContext)
  const id = useId()

  useEffect(() => {
    const removeStep = addStep(id, children)

    return removeStep
  }, [children, addStep, id])

  return null
}
