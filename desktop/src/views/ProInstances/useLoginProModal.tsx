import { BottomActionBar, BottomActionBarError, Form, useStreamingTerminal } from "@/components"
import { useProInstances } from "@/contexts"
import { exists, useFormErrors } from "@/lib"
import { Routes } from "@/routes"
import {
  Box,
  Button,
  Container,
  Divider,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Heading,
  Input,
  InputGroup,
  InputLeftAddon,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalHeader,
  ModalOverlay,
  Tooltip,
  VStack,
  useDisclosure,
} from "@chakra-ui/react"
import { useCallback, useEffect, useMemo, useRef } from "react"
import { SubmitHandler, useForm } from "react-hook-form"
import { useNavigate } from "react-router"
import { ConfigureProviderOptionsForm } from "../Providers/AddProvider"
import { useSetupProvider } from "../Providers/AddProvider/useSetupProvider"

type TFormValues = {
  [FieldName.PRO_URL]: string
  [FieldName.PRO_NAME]: string | undefined
}
const FieldName = {
  PRO_URL: "proURL",
  PRO_NAME: "proName",
} as const

export function useLoginProModal() {
  const { terminal, connectStream } = useStreamingTerminal({ fontSize: "sm" })
  const [[proInstances], { login, disconnect }] = useProInstances()
  const { isOpen, onClose, onOpen } = useDisclosure()
  const { handleSubmit, formState, register, reset } = useForm<TFormValues>({
    mode: "onBlur",
  })
  const containerRef = useRef<HTMLDivElement>(null)
  const onSubmit = useCallback<SubmitHandler<TFormValues>>(
    (data) => {
      login.run({
        url: data[FieldName.PRO_URL],
        name: data[FieldName.PRO_NAME],
        streamListener: connectStream,
      })
    },
    [connectStream, login]
  )

  const {
    state,
    reset: resetSetupProvider,
    completeSetupProvider,
    completeConfigureProvider,
    removeDanglingProviders,
  } = useSetupProvider()

  const { proURLError, proNameError } = useFormErrors(Object.values(FieldName), formState)

  useEffect(() => {
    if (login.status === "success") {
      const providerID = login.provider?.config?.name
      const options = login.provider?.config?.options
      const optionGroups = login.provider?.config?.optionGroups

      if (!exists(providerID)) {
        return
      }
      completeSetupProvider({
        providerID,
        options: options ?? {},
        optionGroups: optionGroups ?? [],
      })
    }
  }, [completeSetupProvider, login.provider, login.status])

  const handleModalClose = useCallback(() => {
    onClose()
    // Make sure to reset on modal close as we'll rarely unmount the hook
    reset({})
    resetSetupProvider()
    login.reset()
    removeDanglingProviders()
    if (state.currentStep !== "done" && state.providerID) {
      disconnect.run({ id: state.providerID })
    }
  }, [
    disconnect,
    login,
    onClose,
    removeDanglingProviders,
    reset,
    resetSetupProvider,
    state.currentStep,
    state.providerID,
  ])

  const areInputsDisabled = useMemo(
    () => login.status === "success" || login.status === "loading",
    [login.status]
  )

  const navigate = useNavigate()
  const completeFlow = useCallback(() => {
    completeConfigureProvider()
    onClose()
    navigate(Routes.WORKSPACE_CREATE)
  }, [completeConfigureProvider, navigate, onClose])

  const modal = useMemo(() => {
    return (
      <Modal
        onClose={handleModalClose}
        isOpen={isOpen}
        closeOnEsc={login.status !== "loading"}
        closeOnOverlayClick={login.status !== "loading"}
        isCentered
        size="4xl"
        scrollBehavior="inside">
        <ModalOverlay />
        <ModalContent overflow="hidden">
          {login.status !== "loading" && <ModalCloseButton />}
          <ModalHeader>Connect to Loft DevPod Pro</ModalHeader>
          <ModalBody overflowX="hidden" overflowY="auto" paddingBottom="0" ref={containerRef}>
            <VStack align="start" spacing="8" paddingX="4" paddingTop="4">
              <Form onSubmit={handleSubmit(onSubmit)} justifyContent="center">
                <Container
                  minHeight="40"
                  maxWidth="container.md"
                  display="flex"
                  flexDirection="row"
                  flexWrap="nowrap"
                  gap="4">
                  <FormControl
                    flexBasis={"70%"}
                    isRequired
                    isInvalid={exists(proURLError)}
                    isDisabled={areInputsDisabled}>
                    <FormLabel>URL</FormLabel>
                    <InputGroup>
                      <InputLeftAddon>https://</InputLeftAddon>
                      <Input
                        type="text"
                        placeholder="my-pro.my-domain.com"
                        {...register(FieldName.PRO_URL, {
                          required: true,
                          validate: {
                            url: (value) => {
                              try {
                                new URL(`https://${value.replace(/^https?:\/\//, "")}`)

                                return true
                              } catch (error) {
                                return "Please enter a valid URL"
                              }
                            },
                            unique: (value) => {
                              const isURLTaken = proInstances?.some(
                                (instance) => instance.url === `https://${value}`
                              )

                              return isURLTaken
                                ? `URL must be unique, an instance with the URL ${value} already exists`
                                : true
                            },
                          },
                        })}
                      />
                    </InputGroup>
                    {proURLError && proURLError.message ? (
                      <FormErrorMessage>{proURLError.message}</FormErrorMessage>
                    ) : (
                      <FormHelperText>
                        Enter a URL to the Loft DevPod Pro instance you intend to connect to. If
                        you&apos;re unsure about it, ask your company administrator or create a new
                        Pro instance on your local machine.
                      </FormHelperText>
                    )}
                  </FormControl>
                  <FormControl
                    flexBasis="33%"
                    isInvalid={exists(proNameError)}
                    isDisabled={areInputsDisabled}>
                    <FormLabel>Instance Name</FormLabel>
                    <InputGroup>
                      <Input
                        type="text"
                        placeholder="Loft"
                        {...register(FieldName.PRO_NAME, {
                          required: false,
                          validate: {
                            unique: (value) => {
                              if (value === undefined) return true
                              const isNameTaken = proInstances?.some(
                                (instance) => instance.id === value
                              )

                              return isNameTaken
                                ? `Name must be unique, an instance named ${value} already exists`
                                : true
                            },
                          },
                        })}
                      />
                    </InputGroup>
                    {proNameError && proNameError.message ? (
                      <FormErrorMessage>{proNameError.message}</FormErrorMessage>
                    ) : (
                      <FormHelperText>Optionally give your instance a name</FormHelperText>
                    )}
                  </FormControl>
                </Container>

                {login.status !== "idle" && state.currentStep === "select-provider" && (
                  <Box width="full" height="10rem">
                    {terminal}
                  </Box>
                )}

                {state.currentStep !== "configure-provider" && (
                  <BottomActionBar isModal>
                    <Tooltip label="Please fill in URL" isDisabled={formState.isValid}>
                      <Button
                        type="submit"
                        variant="primary"
                        isLoading={formState.isSubmitting || login.status === "loading"}
                        isDisabled={!formState.isValid}
                        title="Login">
                        Login
                      </Button>
                    </Tooltip>

                    <BottomActionBarError containerRef={containerRef} error={login.error} />
                  </BottomActionBar>
                )}
              </Form>
              {state.currentStep === "configure-provider" && (
                <>
                  <Divider />
                  <Heading size="md" as="h2">
                    Configure your Pro provider
                  </Heading>

                  <ConfigureProviderOptionsForm
                    isModal
                    isDefault
                    addProvider
                    reuseMachine={false}
                    showBottomActionBar={true}
                    providerID={state.providerID}
                    containerRef={containerRef}
                    onFinish={completeFlow}
                  />
                </>
              )}
            </VStack>
          </ModalBody>
        </ModalContent>
      </Modal>
    )
  }, [
    areInputsDisabled,
    completeFlow,
    formState.isSubmitting,
    formState.isValid,
    handleModalClose,
    handleSubmit,
    isOpen,
    login.error,
    login.status,
    onSubmit,
    proInstances,
    proNameError,
    proURLError,
    register,
    state.currentStep,
    state.providerID,
    terminal,
  ])

  return { modal, handleOpenLogin: onOpen }
}
