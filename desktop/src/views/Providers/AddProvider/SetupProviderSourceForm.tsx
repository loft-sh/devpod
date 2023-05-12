import { CheckIcon } from "@chakra-ui/icons"
import {
  Box,
  Button,
  ButtonGroup,
  Code,
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
  useBreakpointValue,
} from "@chakra-ui/react"
import styled from "@emotion/styled"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { AnimatePresence, motion } from "framer-motion"
import { useCallback, useEffect, useRef, useState } from "react"
import { Controller, ControllerRenderProps, SubmitHandler, useForm } from "react-hook-form"
import { AiOutlinePlusCircle } from "react-icons/ai"
import { FiFolder } from "react-icons/fi"
import { useBorderColor } from "../../../Theme"
import { client } from "../../../client"
import { ErrorMessageBox, ExampleCard } from "../../../components"
import { RECOMMENDED_PROVIDER_SOURCES } from "../../../constants"
import { useProviders } from "../../../contexts"
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
}>
export function SetupProviderSourceForm({
  suggestedProvider,
  reset,
  onFinish,
}: TSetupProviderSourceFormProps) {
  const [[providers], { remove }] = useProviders()
  const [showCustom, setShowCustom] = useState(false)
  const { register, handleSubmit, formState, watch, setValue, control } = useForm<TFormValues>({
    mode: "onBlur",
  })
  const providerSource = watch(FieldName.PROVIDER_SOURCE, "")
  const providerName = watch(FieldName.PROVIDER_NAME, undefined)
  const queryClient = useQueryClient()
  const borderColor = useBorderColor()

  const removeDanglingProviders = useCallback(() => {
    // delete the old selected provider(s)
    const providerIDs = client.providers.popDangling()
    for (const providerID of providerIDs) {
      remove.run({ providerID })
    }
  }, [remove])

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
      removeDanglingProviders()

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
      const options = (await client.providers.getOptions(providerID!)).unwrap()

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
      onFinish(result)
    },
    onError() {
      reset()
    },
  })

  const onSubmit = useCallback<SubmitHandler<TFormValues>>(
    (data) => {
      const providerSource = data[FieldName.PROVIDER_SOURCE].trim()
      const maybeProviderName = data[FieldName.PROVIDER_NAME]?.trim()

      // try to delete dangling providers

      addProvider({ rawProviderSource: providerSource, config: { name: maybeProviderName } })
    },
    [addProvider]
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

  const isLoading = formState.isSubmitting || status === "loading"

  const handleRecommendedProviderClicked = useCallback(
    (sourceName: string) => () => {
      setShowCustom(false)

      const opts = {
        shouldDirty: true,
        shouldValidate: true,
      }
      setValue(FieldName.PROVIDER_SOURCE, sourceName, opts)
      if (providerSource === sourceName) {
        setValue(FieldName.PROVIDER_NAME, undefined, opts)

        return
      }

      reset()
      if (providers?.[sourceName] !== undefined) {
        setValue(FieldName.PROVIDER_NAME, `${sourceName}-${randomString(8)}`, opts)
        removeDanglingProviders()
      } else {
        setValue(FieldName.PROVIDER_NAME, undefined, opts)
        handleSubmit(onSubmit)()
      }
    },
    [handleSubmit, onSubmit, providerSource, providers, removeDanglingProviders, reset, setValue]
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

  const handleSelectFileClicked = useCallback(async () => {
    const selected = await client.selectFromFileYaml()
    if (typeof selected === "string") {
      setValue(FieldName.PROVIDER_SOURCE, selected, {
        shouldDirty: true,
      })
    }
  }, [setValue])

  const exampleCardSize = useBreakpointValue<"md" | "lg">({ base: "md", xl: "lg" })

  const genericProviders = RECOMMENDED_PROVIDER_SOURCES.filter((p) => p.group === "generic")
  const cloudProviders = RECOMMENDED_PROVIDER_SOURCES.filter((p) => p.group === "cloud")

  return (
    <>
      <Form onSubmit={handleSubmit(onSubmit)} spellCheck={false}>
        <Stack spacing={6} width="full" alignItems="center">
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
                    source={source.name}
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
                    source={source.name}
                    isDisabled={isLoading}
                    isSelected={providerSource === source.name}
                    onClick={handleRecommendedProviderClicked(source.name)}
                  />
                ))}
              </HStack>

              <Box paddingX="6" marginInlineStart="0 !important">
                <ExampleCard
                  size={exampleCardSize}
                  imageNode={<Icon as={AiOutlinePlusCircle} color={"primary.500"} />}
                  isSelected={showCustom}
                  isDisabled={isLoading}
                  onClick={() => {
                    setShowCustom(!showCustom)
                    setValue(FieldName.PROVIDER_SOURCE, "", {
                      shouldDirty: true,
                    })
                  }}
                />
              </Box>
            </HStack>

            {showCustom && (
              <Box marginTop={"10px"} maxWidth={"700px"}>
                <FormLabel>Source</FormLabel>
                <HStack spacing={0} justifyContent={"center"}>
                  <Input
                    spellCheck={false}
                    placeholder="Enter provider source"
                    borderTopRightRadius={0}
                    borderBottomRightRadius={0}
                    type="text"
                    {...register(FieldName.PROVIDER_SOURCE, { required: true })}
                  />
                  <Button
                    leftIcon={<Icon as={FiFolder} />}
                    borderTopLeftRadius={0}
                    borderBottomLeftRadius={0}
                    borderTop={"1px solid white"}
                    borderRight={"1px solid white"}
                    borderBottom={"1px solid white"}
                    borderColor={"gray.200"}
                    height={"35px"}
                    flex={"0 0 140px"}
                    onClick={handleSelectFileClicked}>
                    Select File
                  </Button>
                </HStack>
                {providerSourceError && providerSourceError.message ? (
                  <FormErrorMessage>{providerSourceError.message}</FormErrorMessage>
                ) : (
                  <FormHelperText>
                    Can either be a URL or local path to a <Code>provider</Code> file, or a github
                    repo in the form of <Code>my-org/my-repo</Code>
                  </FormHelperText>
                )}
              </Box>
            )}
          </FormControl>

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
