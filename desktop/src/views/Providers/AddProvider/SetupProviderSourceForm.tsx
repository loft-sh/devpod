import { CheckIcon } from "@chakra-ui/icons"
import {
  Box,
  Button,
  ButtonGroup,
  Code,
  Container,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  HStack,
  Icon,
  Input,
  InputGroup,
  InputProps,
  InputRightElement,
  Stack,
  Text,
  useBreakpointValue,
} from "@chakra-ui/react"
import styled from "@emotion/styled"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { AnimatePresence, motion } from "framer-motion"
import { useCallback, useEffect, useRef, useState } from "react"
import { Controller, ControllerRenderProps, SubmitHandler, useForm } from "react-hook-form"
import { FiFolder } from "react-icons/fi"
import { useBorderColor } from "../../../Theme"
import { client } from "../../../client"
import { ErrorMessageBox, ExampleCard } from "../../../components"
import { RECOMMENDED_PROVIDER_SOURCES } from "../../../constants"
import { useProviders } from "../../../contexts"
import { CustomSvg } from "../../../images"
import { exists, isError, randomString, useFormErrors } from "../../../lib"
import { QueryKeys } from "../../../queryKeys"
import {
  TAddProviderConfig,
  TProviderID,
  TProviderOptionGroup,
  TProviderOptions,
  TWithProviderID,
} from "../../../types"
import { FieldName, TFormValues } from "./types"

const Form = styled.form`
  width: 100%;
  display: flex;
  flex-flow: column nowrap;
  justify-content: center;
`
const ALLOWED_NAMES_REGEX = /^[a-z0-9\\-]+$/

type TSetupProviderSourceFormProps = Readonly<{
  suggestedProvider?: TProviderID
  reset: () => void
  onFinish: (
    result: TWithProviderID &
      Readonly<{ options: TProviderOptions; optionGroups: TProviderOptionGroup[] }>
  ) => void
  removeDanglingProviders: VoidFunction
}>
export function SetupProviderSourceForm({
  suggestedProvider,
  reset,
  onFinish,
  removeDanglingProviders,
}: TSetupProviderSourceFormProps) {
  const [[providers]] = useProviders()
  const [showCustom, setShowCustom] = useState(false)
  const { handleSubmit, formState, watch, setValue, control } = useForm<TFormValues>({
    mode: "onBlur",
  })
  const providerSource = watch(FieldName.PROVIDER_SOURCE, "")
  const providerName = watch(FieldName.PROVIDER_NAME, undefined)
  const queryClient = useQueryClient()
  const borderColor = useBorderColor()

  const {
    mutate: addProvider,
    status,
    error,
    reset: resetAddProvider,
  } = useMutation({
    mutationFn: async ({
      rawProviderSource,
      config,
    }: Readonly<{
      rawProviderSource: string
      config: TAddProviderConfig
    }>) => {
      // check if provider exists and is not initialized
      const providerID = config.name || (await client.providers.newID(rawProviderSource)).unwrap()
      if (!providerID) {
        throw new Error(`Couldn't find provider id`)
      }

      // list all providers
      let providers = (await client.providers.listAll()).unwrap()
      if (providers?.[providerID]) {
        if (!providers[providerID]?.state?.initialized) {
          ;(await client.providers.remove(providerID)).unwrap()
        } else {
          throw new Error(
            `Provider with name ${providerID} already exists, please choose a different name`
          )
        }
      }

      // add provider
      ;(await client.providers.add(rawProviderSource, config)).unwrap()

      // get options
      let options: TProviderOptions | undefined
      try {
        options = (await client.providers.getOptions(providerID!)).unwrap()
      } catch (e) {
        ;(await client.providers.remove(providerID)).unwrap()
        throw e
      }

      // check if provider could be added
      providers = (await client.providers.listAll()).unwrap()
      if (!providers?.[providerID!]) {
        throw new Error(`Provider ${providerID} couldn't be found`)
      }

      return {
        providerID: providerID!,
        options: options!,
        optionGroups: providers[providerID!]?.config?.optionGroups || [],
      }
    },
    onSuccess(result) {
      queryClient.invalidateQueries(QueryKeys.PROVIDERS)
      setValue(FieldName.PROVIDER_NAME, undefined, { shouldDirty: true })
      setShowCustom(false)
      onFinish(result)
    },
    onError() {
      reset()
    },
  })

  const onSubmit = useCallback<SubmitHandler<TFormValues>>(
    async (data) => {
      const providerSource = data[FieldName.PROVIDER_SOURCE].trim()
      const maybeProviderName = data[FieldName.PROVIDER_NAME]?.trim()

      const opts = {
        shouldDirty: true,
        shouldValidate: true,
      }

      const providerIDRes = await client.providers.newID(providerSource)
      let preferredProviderName: string | undefined
      if (providerIDRes.ok) {
        preferredProviderName = providerIDRes.val
      }

      removeDanglingProviders()
      // custom name taken
      if (maybeProviderName !== undefined && providers?.[maybeProviderName] !== undefined) {
        setValue(FieldName.PROVIDER_NAME, `${maybeProviderName}-${randomString(8)}`, opts)
        // preferred ID available
      } else if (maybeProviderName === undefined && preferredProviderName !== undefined) {
        // preferred ID taken
        if (providers?.[preferredProviderName] !== undefined) {
          setValue(FieldName.PROVIDER_NAME, `${preferredProviderName}-${randomString(8)}`, opts)
        } else {
          // preferred ID available
          setValue(FieldName.PROVIDER_NAME, undefined, opts)
          addProvider({
            rawProviderSource: providerSource,
            config: { name: preferredProviderName },
          })
        }
      } else {
        setValue(FieldName.PROVIDER_NAME, undefined, opts)
        addProvider({
          rawProviderSource: providerSource,
          config: { name: maybeProviderName ?? preferredProviderName },
        })
      }
    },
    [addProvider, providers, removeDanglingProviders, setValue]
  )

  useEffect(() => {
    const watchProviderSource = watch(() => {
      // Reset the provider mutation if the source changes after we ran into an error
      if (status === "error") {
        resetAddProvider()
      }
    })

    return () => watchProviderSource.unsubscribe()
  }, [watch, status, resetAddProvider])

  const { providerSourceError, providerNameError } = useFormErrors(
    Object.values(FieldName),
    formState
  )

  const handleRecommendedProviderClicked = useCallback(
    (sourceName: string) => () => {
      setShowCustom(false)

      const opts = {
        shouldDirty: true,
        shouldValidate: true,
      }
      setValue(FieldName.PROVIDER_SOURCE, sourceName, opts)
      setValue(FieldName.PROVIDER_NAME, undefined, opts)
      if (providerSource === sourceName) {
        return
      }

      reset()
      handleSubmit(onSubmit)()
    },
    [handleSubmit, onSubmit, providerSource, reset, setValue]
  )

  const suggestedProviderLock = useRef(true)
  // handle provider suggestion
  useEffect(() => {
    if (
      suggestedProvider &&
      suggestedProvider !== "" &&
      (!exists(providerSource) || providerSource === "") &&
      suggestedProviderLock.current
    ) {
      suggestedProviderLock.current = false
      setValue(FieldName.PROVIDER_SOURCE, suggestedProvider, {
        shouldDirty: true,
        shouldValidate: true,
      })
      handleRecommendedProviderClicked(suggestedProvider)()
    }
  }, [handleRecommendedProviderClicked, providerSource, setValue, suggestedProvider])

  const handleCustomProviderClicked = useCallback(() => {
    setShowCustom(true)
    const opts = {
      shouldDirty: true,
      shouldValidate: true,
    }
    setValue(FieldName.PROVIDER_SOURCE, "", opts)
    setValue(FieldName.PROVIDER_NAME, undefined, opts)
    removeDanglingProviders()
    reset()
  }, [removeDanglingProviders, reset, setValue])

  const isLoading = formState.isSubmitting || status === "loading"
  const exampleCardSize = useBreakpointValue<"md" | "lg">({ base: "md", xl: "lg" })

  const genericProviders = RECOMMENDED_PROVIDER_SOURCES.filter((p) => p.group === "generic")
  const cloudProviders = RECOMMENDED_PROVIDER_SOURCES.filter((p) => p.group === "cloud")

  return (
    <>
      <Form onSubmit={handleSubmit(onSubmit)} spellCheck={false}>
        <Stack spacing={6} width="full" alignItems="center" paddingBottom={8}>
          <FormControl
            isRequired
            isInvalid={exists(providerSourceError)}
            display="flex"
            justifyContent="center">
            <HStack marginY="8" spacing="0" width="fit-content">
              <HStack
                paddingX="6"
                spacing="4"
                height="full"
                borderRightWidth="thin"
                borderColor={borderColor}>
                {genericProviders.map((source) => (
                  <ExampleCard
                    size={exampleCardSize}
                    key={source.name}
                    image={source.image}
                    name={source.name}
                    isSelected={providerSource === source.name}
                    isDisabled={isLoading}
                    onClick={handleRecommendedProviderClicked(source.name)}
                  />
                ))}
              </HStack>

              <HStack
                paddingX="6"
                spacing="4"
                height="full"
                borderRightWidth="thin"
                borderColor={borderColor}>
                {cloudProviders.map((source) => (
                  <ExampleCard
                    size={exampleCardSize}
                    key={source.name}
                    image={source.image}
                    name={source.name}
                    isDisabled={isLoading}
                    isSelected={providerSource === source.name}
                    onClick={handleRecommendedProviderClicked(source.name)}
                  />
                ))}
              </HStack>

              <Box paddingX="6" marginInlineStart="0 !important">
                <ExampleCard
                  size={exampleCardSize}
                  name="custom"
                  image={CustomSvg}
                  isSelected={showCustom}
                  isDisabled={isLoading}
                  onClick={handleCustomProviderClicked}
                />
              </Box>
            </HStack>
          </FormControl>

          <Container color="gray.600" maxWidth="container.md">
            {showCustom && (
              <FormControl isRequired isInvalid={exists(providerSourceError)}>
                <FormLabel>Source</FormLabel>
                <Controller
                  name={FieldName.PROVIDER_SOURCE}
                  rules={{ required: true }}
                  control={control}
                  render={({ field }) => (
                    <CustomProviderInput
                      field={field}
                      isInvalid={exists(providerSourceError)}
                      onAccept={handleSubmit(onSubmit)}
                    />
                  )}
                />
                {providerSourceError && providerSourceError.message ? (
                  <FormErrorMessage>{providerSourceError.message}</FormErrorMessage>
                ) : (
                  <FormHelperText>
                    Can either be a URL or local path to a <Code>provider</Code> file, or a github
                    repo in the form of <Code>my-org/my-repo</Code>
                  </FormHelperText>
                )}
              </FormControl>
            )}

            {!formState.isDirty && (
              <>
                <Text fontWeight="bold">Choose your provider</Text>
                <Text marginBottom="8">
                  Providers determine how and where your workspaces run. They connect to the cloud
                  platform - or local environment - of your choice and spin up your workspaces. You
                  can choose from a number of pre-built providers, or connect your own.
                </Text>
              </>
            )}
          </Container>

          <AnimatePresence>
            {exists(providerName) && (
              <FormControl
                maxWidth={{ base: "3xl", xl: "4xl" }}
                as={motion.div}
                initial={{ height: 0, overflow: "hidden" }}
                animate={{ height: "auto", overflow: "revert" }}
                exit={{ height: 0, overflow: "hidden" }}
                isInvalid={exists(providerNameError)}>
                <FormLabel>Name</FormLabel>
                <Controller
                  name={FieldName.PROVIDER_NAME}
                  control={control}
                  rules={{
                    pattern: {
                      value: ALLOWED_NAMES_REGEX,
                      message: "Name can only contain letters, numbers and -",
                    },
                    validate: {
                      unique: (value) => {
                        if (value === undefined) return true
                        if (value === "") return "Name cannot be empty"

                        return providers?.[value] === undefined ? true : "Name must be unique"
                      },
                    },
                    maxLength: { value: 48, message: "Name cannot be longer than 48 characters" },
                  }}
                  render={({ field }) => (
                    <CustomNameInput
                      field={field}
                      onAccept={handleSubmit(onSubmit)}
                      isInvalid={exists(providerNameError)}
                    />
                  )}
                />
                {exists(providerNameError) ? (
                  <FormErrorMessage>{providerNameError.message ?? "Error"}</FormErrorMessage>
                ) : (
                  <FormHelperText>
                    Please give your provider a different name from the one specified in its{" "}
                    <Code>provider.yaml</Code>
                  </FormHelperText>
                )}
              </FormControl>
            )}
          </AnimatePresence>

          {status === "error" && isError(error) && <ErrorMessageBox error={error} />}
          {isLoading && <LoadingProvider name={providerName ?? providerSource} />}
        </Stack>
      </Form>
    </>
  )
}

type TCustomNameInputProps = Readonly<{
  field: ControllerRenderProps<TFormValues, (typeof FieldName)["PROVIDER_NAME"]>
  isInvalid: boolean
  onAccept: () => void
}> &
  InputProps
function CustomNameInput({ field, isInvalid, onAccept }: TCustomNameInputProps) {
  return (
    <InputGroup>
      <Input
        size="md"
        type="text"
        placeholder="Custom provider name"
        spellCheck={false}
        value={field.value}
        onBlur={field.onBlur}
        onChange={(e) => field.onChange(e.target.value)}
      />
      <InputRightElement width="24">
        <ButtonGroup>
          <Button
            variant="outline"
            aria-label="Save new name"
            colorScheme="green"
            isDisabled={isInvalid}
            leftIcon={<CheckIcon boxSize="4" />}
            onClick={() => onAccept()}>
            Save
          </Button>
        </ButtonGroup>
      </InputRightElement>
    </InputGroup>
  )
}

type TCustomProviderInputProps = Readonly<{
  field: ControllerRenderProps<TFormValues, (typeof FieldName)["PROVIDER_SOURCE"]>
  isInvalid: boolean
  onAccept: () => void
}> &
  InputProps
function CustomProviderInput({ field, isInvalid, onAccept }: TCustomProviderInputProps) {
  const handleSelectFileClicked = useCallback(async () => {
    const selected = await client.selectFromFileYaml()
    if (typeof selected === "string") {
      field.onChange(selected)
      field.onBlur()
    }
  }, [field])

  return (
    <InputGroup>
      <Input
        spellCheck={false}
        placeholder="loft-sh/devpod-provider-terraform"
        type="text"
        value={field.value}
        onBlur={field.onBlur}
        onChange={(e) => field.onChange(e.target.value)}
      />
      <InputRightElement width="64">
        <Button
          variant="outline"
          aria-label="Continue"
          colorScheme="green"
          isDisabled={isInvalid}
          marginRight="2"
          marginLeft="3"
          leftIcon={<CheckIcon boxSize="4" />}
          onClick={() => onAccept()}>
          Continue
        </Button>
        <Button
          leftIcon={<Icon as={FiFolder} />}
          borderWidth="thin"
          borderTopRightRadius="md"
          borderBottomRightRadius="md"
          borderTopLeftRadius="0"
          borderBottomLeftRadius="0"
          borderColor={"gray.200"}
          marginRight="-2px"
          onClick={handleSelectFileClicked}
          height="calc(100% - 2px)">
          Select File
        </Button>
      </InputRightElement>
    </InputGroup>
  )
}

function LoadingProvider({ name }: Readonly<{ name: string | undefined }>) {
  return (
    <HStack marginTop="2" justifyContent="center" alignItems="center" color="gray.600">
      <Text fontWeight="medium">Loading {name}</Text>
      <Box as="svg" height="3" marginInlineStart="0 !important" width="8" viewBox="0 0 48 30">
        <circle fill="currentColor" stroke="none" cx="6" cy="24" r="6">
          <animateTransform
            attributeName="transform"
            dur="1s"
            type="translate"
            values="0 0; 0 -12; 0 0; 0 0; 0 0; 0 0"
            repeatCount="indefinite"
            begin="0"
          />
        </circle>
        <circle fill="currentColor" stroke="none" cx="24" cy="24" r="6">
          <animateTransform
            id="op"
            attributeName="transform"
            dur="1s"
            type="translate"
            values="0 0; 0 -12; 0 0; 0 0; 0 0; 0 0"
            repeatCount="indefinite"
            begin="0.3s"
          />
        </circle>
        <circle fill="currentColor" stroke="none" cx="42" cy="24" r="6">
          <animateTransform
            id="op"
            attributeName="transform"
            dur="1s"
            type="translate"
            values="0 0; 0 -12; 0 0; 0 0; 0 0; 0 0"
            repeatCount="indefinite"
            begin="0.6s"
          />
        </circle>
      </Box>
    </HStack>
  )
}
