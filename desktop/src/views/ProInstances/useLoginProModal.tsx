import { BottomActionBar, BottomActionBarError, Form, useStreamingTerminal } from "@/components"
import { useProInstances, useProviders } from "@/contexts"
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
import { ALLOWED_NAMES_REGEX } from "../Providers/AddProvider/helpers"

type TFormValues = {
  [FieldName.PRO_HOST]: string
  [FieldName.PROVIDER_NAME]: string | undefined
}
const FieldName = {
  PRO_HOST: "proURL",
  PROVIDER_NAME: "providerName",
} as const

export function useLoginProModal() {
  const { terminal, connectStream, clear } = useStreamingTerminal({ fontSize: "sm" })
  const [[proInstances], { login, disconnect }] = useProInstances()
  const [[providers]] = useProviders()
  const { isOpen, onClose, onOpen } = useDisclosure()
  const { handleSubmit, formState, register, reset } = useForm<TFormValues>({
    mode: "onBlur",
  })
  const containerRef = useRef<HTMLDivElement>(null)
  const onSubmit = useCallback<SubmitHandler<TFormValues>>(
    (data) => {
      clear()
      login.run({
        host: data[FieldName.PRO_HOST],
        providerName: data[FieldName.PROVIDER_NAME],
        streamListener: connectStream,
      })
    },
    [connectStream, login, clear]
  )

  const {
    state,
    reset: resetSetupProvider,
    completeSetupProvider,
    completeConfigureProvider,
    removeDanglingProviders,
  } = useSetupProvider()

  const { proURLError, providerNameError } = useFormErrors(Object.values(FieldName), formState)

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
                        {...register(FieldName.PRO_HOST, {
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
                              const isHostTaken = proInstances?.some(
                                (instance) => instance.host === value
                              )

                              return isHostTaken
                                ? `URL must be unique, an instance with the URL https://${value} already exists`
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
                    isInvalid={exists(providerNameError)}
                    isDisabled={areInputsDisabled}>
                    <FormLabel>Provider Name</FormLabel>
                    <InputGroup>
                      <Input
                        type="text"
                        placeholder="devpod-pro"
                        {...register(FieldName.PROVIDER_NAME, {
                          required: false,
                          pattern: {
                            value: ALLOWED_NAMES_REGEX,
                            message: "Name can only contain lowercase letters, numbers and -",
                          },
                          validate: {
                            unique: (value) => {
                              if (value === undefined) return true
                              const isNameTaken = providers?.[value] !== undefined

                              return isNameTaken
                                ? `Name must be unique, a provider named ${value} already exists`
                                : true
                            },
                          },
                          maxLength: {
                            value: 48,
                            message: "Name cannot be longer than 48 characters",
                          },
                        })}
                      />
                    </InputGroup>
                    {providerNameError && providerNameError.message ? (
                      <FormErrorMessage>{providerNameError.message}</FormErrorMessage>
                    ) : (
                      <FormHelperText>Optionally give the pro provider a name</FormHelperText>
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
    providerNameError,
    proURLError,
    providers,
    register,
    state.currentStep,
    state.providerID,
    terminal,
  ])

  return { modal, handleOpenLogin: onOpen }
}
