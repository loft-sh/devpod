import { Form } from "@/components/Form"
import { CheckIcon } from "@chakra-ui/icons"
import {
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
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Stack,
  Text,
  useBreakpointValue,
  useColorMode,
  useColorModeValue,
} from "@chakra-ui/react"
import { useQueryClient } from "@tanstack/react-query"
import { AnimatePresence, motion } from "framer-motion"
import { useCallback, useEffect, useRef, useState } from "react"
import {
  Controller,
  ControllerRenderProps,
  SetValueConfig,
  SubmitHandler,
  useForm,
} from "react-hook-form"
import { FiFolder } from "react-icons/fi"
import { client } from "../../../client"
import { ErrorMessageBox, ExampleCard } from "../../../components"
import { RECOMMENDED_PROVIDER_SOURCES } from "../../../constants"
import { useProviders } from "../../../contexts"
import { Stack3D } from "../../../icons"
import { CommunitySvg, CustomSvg } from "../../../images"
import { exists, isError, randomString, useFormErrors } from "../../../lib"
import { QueryKeys } from "../../../queryKeys"
import { TCommunityProvider, TProviderID } from "../../../types"
import { useCommunityContributions } from "../../../useCommunityContributions"
import { LoadingProviderIndicator } from "./LoadingProviderIndicator"
import { FieldName, TFormValues, TSetupProviderResult } from "./types"
import { useAddProvider } from "./useAddProvider"

const ALLOWED_NAMES_REGEX = /^[a-z0-9\\-]+$/
const DEFAULT_VAL_OPTS: SetValueConfig = {
  shouldDirty: true,
  shouldValidate: true,
}

type TSetupProviderSourceFormProps = Readonly<{
  suggestedProvider?: TProviderID
  reset: () => void
  onFinish: (result: TSetupProviderResult) => void
  removeDanglingProviders: VoidFunction
}>
export function SetupProviderSourceForm({
  suggestedProvider,
  reset,
  onFinish,
  removeDanglingProviders,
}: TSetupProviderSourceFormProps) {
  const [[providers]] = useProviders()
  const { contributions } = useCommunityContributions()
  const [showCustom, setShowCustom] = useState({
    manual: false,
    community: false,
  })
  const { handleSubmit, formState, watch, setValue, control } = useForm<TFormValues>({
    mode: "onBlur",
  })
  const providerSource = watch(FieldName.PROVIDER_SOURCE, "")
  const providerName = watch(FieldName.PROVIDER_NAME, undefined)
  const queryClient = useQueryClient()
  const borderColor = useColorModeValue("gray.200", "gray.600")
  const hoverBackgroundColor = useColorModeValue("gray.50", "gray.800")
  const { colorMode } = useColorMode()

  const {
    mutate: addProvider,
    status,
    error,
    reset: resetAddProvider,
  } = useAddProvider({
    onSuccess(result) {
      queryClient.invalidateQueries(QueryKeys.PROVIDERS)
      setValue(FieldName.PROVIDER_NAME, undefined, { shouldDirty: true })
      setShowCustom({ manual: false, community: false })
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

      const providerIDRes = await client.providers.newID(providerSource)
      let preferredProviderName: string | undefined
      if (providerIDRes.ok) {
        preferredProviderName = providerIDRes.val
      }

      removeDanglingProviders()
      // custom name taken
      if (maybeProviderName !== undefined && providers?.[maybeProviderName] !== undefined) {
        setValue(
          FieldName.PROVIDER_NAME,
          `${maybeProviderName}-${randomString(8)}`,
          DEFAULT_VAL_OPTS
        )
        // preferred ID available
      } else if (maybeProviderName === undefined && preferredProviderName !== undefined) {
        // preferred ID taken
        if (providers?.[preferredProviderName] !== undefined) {
          setValue(
            FieldName.PROVIDER_NAME,
            `${preferredProviderName}-${randomString(8)}`,
            DEFAULT_VAL_OPTS
          )
        } else {
          // preferred ID available
          setValue(FieldName.PROVIDER_NAME, undefined, DEFAULT_VAL_OPTS)
          addProvider({
            rawProviderSource: providerSource,
            config: { name: preferredProviderName },
          })
        }
      } else {
        setValue(FieldName.PROVIDER_NAME, undefined, DEFAULT_VAL_OPTS)
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
      setShowCustom({ manual: false, community: false })
      setValue(FieldName.PROVIDER_SOURCE, sourceName, DEFAULT_VAL_OPTS)
      setValue(FieldName.PROVIDER_NAME, undefined, DEFAULT_VAL_OPTS)
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
    setShowCustom({ manual: true, community: false })
    setValue(FieldName.PROVIDER_SOURCE, "", DEFAULT_VAL_OPTS)
    setValue(FieldName.PROVIDER_NAME, undefined, DEFAULT_VAL_OPTS)
    removeDanglingProviders()
    reset()
  }, [removeDanglingProviders, reset, setValue])

  const handleCommunityProviderClicked = useCallback(
    (communityProvider: TCommunityProvider) => {
      setShowCustom({ manual: false, community: true })
      let source = communityProvider.repository
      // Github-hosted providers are special, the CLI expects them to be passed in without the `https://` prefix
      if (source.includes("github.com")) {
        source = communityProvider.repository.replace("https://", "")
      }

      setValue(FieldName.PROVIDER_SOURCE, source, DEFAULT_VAL_OPTS)
      setValue(FieldName.PROVIDER_NAME, undefined, DEFAULT_VAL_OPTS)
      removeDanglingProviders()
      reset()
      handleSubmit(onSubmit)()
    },
    [handleSubmit, onSubmit, removeDanglingProviders, reset, setValue]
  )

  const isLoading = formState.isSubmitting || status === "loading"
  const exampleCardSize = useBreakpointValue<"md" | "lg">({ base: "md", xl: "lg" })

  const genericProviders = RECOMMENDED_PROVIDER_SOURCES.filter((p) => p.group === "generic")
  const cloudProviders = RECOMMENDED_PROVIDER_SOURCES.filter((p) => p.group === "cloud")
  const communityProviders = contributions?.providers

  return (
    <>
      <Form onSubmit={handleSubmit(onSubmit)}>
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
                    image={
                      colorMode == "dark" && source.imageDarkMode
                        ? source.imageDarkMode
                        : source.image
                    }
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
                    image={
                      colorMode == "dark" && source.imageDarkMode
                        ? source.imageDarkMode
                        : source.image
                    }
                    name={source.name}
                    isDisabled={isLoading}
                    isSelected={providerSource === source.name}
                    onClick={handleRecommendedProviderClicked(source.name)}
                  />
                ))}
              </HStack>

              <HStack height="full" paddingX="6">
                <Menu placement="left">
                  <MenuButton
                    as={Button}
                    isDisabled={isLoading || !communityProviders || communityProviders.length === 0}
                    _disabled={{ opacity: 1, cursor: "not-allowed" }}
                    _hover={{ bg: hoverBackgroundColor }}
                    variant="ghost"
                    width="fit-content"
                    height="fit-content"
                    paddingInline="0 !important">
                    <ExampleCard
                      size={exampleCardSize}
                      name="community"
                      image={CommunitySvg}
                      isSelected={showCustom.community}
                      isDisabled={
                        isLoading || !communityProviders || communityProviders.length === 0
                      }
                    />
                  </MenuButton>
                  <MenuList overflowY="auto" maxHeight="72">
                    {communityProviders
                      ?.map(mapCommunityProviderInfo)
                      .filter((x): x is NonNullable<typeof x> => x !== undefined)
                      .sort(sortCommunityProviderInfo)
                      .map((info) => (
                        <MenuItem
                          key={info.repository}
                          onClick={() => handleCommunityProviderClicked(info)}>
                          <HStack>
                            <Stack3D boxSize="4" />
                            {typeof info.title === "string" ? (
                              <Text>{info.title}</Text>
                            ) : (
                              <Text>
                                {info.title.org}/<b>{info.title.name}</b>
                              </Text>
                            )}
                          </HStack>
                        </MenuItem>
                      ))}
                  </MenuList>
                </Menu>

                <ExampleCard
                  size={exampleCardSize}
                  name="custom"
                  image={CustomSvg}
                  isSelected={showCustom.manual}
                  isDisabled={isLoading}
                  onClick={handleCustomProviderClicked}
                />
              </HStack>
            </HStack>
          </FormControl>

          <Container color="gray.700" _dark={{ color: "gray.200" }} maxWidth="container.md">
            {showCustom.manual && (
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
          {isLoading && (
            <LoadingProviderIndicator label={`Loading ${providerName ?? providerSource}`} />
          )}
        </Stack>
      </Form>
    </>
  )
}

type TCustomNameInputProps = Readonly<{
  field: ControllerRenderProps<TFormValues, (typeof FieldName)["PROVIDER_NAME"]>
  isInvalid: boolean
  onAccept: () => void
  isDisabled?: boolean
}> &
  InputProps
export function CustomNameInput({ field, isInvalid, onAccept, isDisabled }: TCustomNameInputProps) {
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
            isDisabled={isInvalid || field.value === "" || isDisabled}
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
  const backgroundColor = useColorModeValue("white", "black")
  const handleSelectFileClicked = useCallback(async () => {
    const selected = await client.selectFileYaml()
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
          backgroundColor={backgroundColor}
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

type TCommunityProviderInfo = Readonly<{
  title: string | { org: string; name: string }
  repository: string
}>
function mapCommunityProviderInfo(
  communityProvider: TCommunityProvider
): TCommunityProviderInfo | undefined {
  const repo = communityProvider.repository
  try {
    const url = new URL(repo)
    const segments = url.pathname.split("/").filter((s) => s !== "")

    // probably $ORG/$REPO
    if (segments.length === 2 && segments[0] !== undefined && segments[1] !== undefined) {
      return { title: { org: segments[0], name: stripDevpodPrefix(segments[1]) }, repository: repo }
    }

    const last = segments.pop()
    if (last !== undefined) {
      return { title: stripDevpodPrefix(last), repository: repo }
    }

    return undefined
  } catch (e) {
    console.error(`Unable to convert "${repo}" to URL: ${e}`)

    return undefined
  }
}

function stripDevpodPrefix(rawCommunityProvider: string): string {
  return rawCommunityProvider.replace("devpod-provider-", "")
}

function sortCommunityProviderInfo(a: TCommunityProviderInfo, b: TCommunityProviderInfo): number {
  if (typeof a.title === "string" && typeof b.title === "string") {
    return a.title > b.title ? 1 : -1
  }

  if (typeof a.title === "string") return 1
  if (typeof b.title === "string") return -1

  return a.title.name > b.title.name ? 1 : -1
}
