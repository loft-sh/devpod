import {
  Box,
  Button,
  Code,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Icon,
  Input,
  SimpleGrid,
  Stack,
  VStack,
} from "@chakra-ui/react"
import styled from "@emotion/styled"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useCallback, useDeferredValue, useEffect, useMemo, useState } from "react"
import { SubmitHandler, useForm } from "react-hook-form"
import { client } from "../../../client"
import { CollapsibleSection, ErrorMessageBox } from "../../../components"
import { exists, isEmpty, isError, useFormErrors } from "../../../lib"
import { QueryKeys } from "../../../queryKeys"
import DockerPng from "../../../images/docker.png"
import GCloudSvg from "../../../images/gcloud.svg"
import AWSSvg from "../../../images/aws.svg"
import AzureSvg from "../../../images/azure.svg"
import DigitalOceanSvg from "../../../images/digitalocean.svg"
import KubernetesSvg from "../../../images/kubernetes.svg"
import { AiOutlinePlusCircle } from "react-icons/ai"
import {
  TAddProviderConfig,
  TProviderOptionGroup,
  TProviderOptions,
  TProviders,
  TWithProviderID,
} from "../../../types"
import { FieldName, TFormValues } from "./types"
import { RecommendedProviderCard } from "./RecommendedProviderCard"
import { UseFormSetValue } from "react-hook-form/dist/types/form"
import { TSetupProviderState } from "./useSetupProvider"

const Form = styled.form`
  width: 100%;
`
const ALLOWED_NAMES_REGEX = /^[a-z0-9\\.\\-]+$/

type TSetupProviderSourceFormProps = Readonly<{
  state: TSetupProviderState
  onReset: () => void
  onFinish: (
    result: TWithProviderID &
      Readonly<{ options: TProviderOptions; optionGroups: TProviderOptionGroup[] }>
  ) => void
}>
export function SetupProviderSourceForm({
  state,
  onFinish,
  onReset,
}: TSetupProviderSourceFormProps) {
  const [providers, setProviders] = useState<TProviders | undefined>()
  useEffect(() => {
    ;(async () => {
      setProviders((await client.providers.listAll()).unwrap())
    })()
  }, [])
  const [showCustom, setShowCustom] = useState(false)
  const { register, handleSubmit, formState, watch, setValue } = useForm<TFormValues>({
    mode: "onBlur",
  })
  const providerSource = watch(FieldName.PROVIDER_SOURCE, "")
  const deferredProviderSource = useDeferredValue(providerSource)

  const queryClient = useQueryClient()
  const { data: suggestedProviderName } = useQuery({
    queryKey: ["providerNameSuggestion", deferredProviderSource],
    queryFn: async () => {
      if (!deferredProviderSource) {
        return ""
      }

      return (await client.providers.newID(deferredProviderSource)).unwrap()
    },
    onSuccess(suggestedName) {
      setValue(FieldName.PROVIDER_NAME, suggestedName, {
        shouldDirty: false,
        shouldTouch: false,
        shouldValidate: true,
      })
    },
    enabled: !isEmpty(deferredProviderSource),
  })

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
      if (state.currentStep !== 1) {
        onReset()
      }

      ;(await client.providers.add(rawProviderSource, config)).unwrap()
      const providerID = (await client.providers.newID(rawProviderSource)).unwrap()
      const options = (await client.providers.getOptions(providerID!)).unwrap()
      const providers = (await client.providers.listAll()).unwrap()
      if (!providers?.[providerID!]) {
        throw `Provider ${providerID} couldn't be found`
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
  })

  const onSubmit = useCallback<SubmitHandler<TFormValues>>(
    (data) => {
      const providerSource = data[FieldName.PROVIDER_SOURCE].trim()
      const maybeProviderName = data[FieldName.PROVIDER_NAME]?.trim()

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

  const isSubmitDisabled = useMemo(() => {
    return (
      status === "error" ||
      !formState.dirtyFields[FieldName.PROVIDER_SOURCE] ||
      formState.isSubmitting
    )
  }, [formState.dirtyFields, formState.isSubmitting, status])

  const wrappedSetValue: UseFormSetValue<TFormValues> = (a, b, c) => {
    setShowCustom(false)
    setValue(a, b as any, c)
  }

  return (
    <Form onSubmit={handleSubmit(onSubmit)} spellCheck={false}>
      <Stack spacing={6} width="full">
        <FormControl isRequired isInvalid={exists(providerSourceError)}>
          <SimpleGrid
            spacing={4}
            templateColumns="repeat(auto-fill, minmax(120px, 1fr))"
            marginTop={"10px"}>
            {!providers?.["docker"] && (
              <RecommendedProviderCard
                image={DockerPng}
                source={"docker"}
                currentSource={providerSource}
                setValue={wrappedSetValue}
              />
            )}
            {!providers?.["aws"] && (
              <RecommendedProviderCard
                image={AWSSvg}
                source={"aws"}
                currentSource={providerSource}
                setValue={wrappedSetValue}
              />
            )}
            {!providers?.["gcloud"] && (
              <RecommendedProviderCard
                image={GCloudSvg}
                source={"gcloud"}
                currentSource={providerSource}
                setValue={wrappedSetValue}
              />
            )}
            {!providers?.["azure"] && (
              <RecommendedProviderCard
                image={AzureSvg}
                source={"azure"}
                currentSource={providerSource}
                setValue={wrappedSetValue}
              />
            )}
            {!providers?.["digitalocean"] && (
              <RecommendedProviderCard
                image={DigitalOceanSvg}
                source={"digitalocean"}
                currentSource={providerSource}
                setValue={wrappedSetValue}
              />
            )}
            {!providers?.["kubernetes"] && (
              <RecommendedProviderCard
                image={KubernetesSvg}
                source={"kubernetes"}
                currentSource={providerSource}
                setValue={wrappedSetValue}
              />
            )}
            <RecommendedProviderCard
              imageNode={<Icon as={AiOutlinePlusCircle} fontSize={"64px"} color={"primary.500"} />}
              selected={showCustom}
              currentSource={providerSource}
              onClick={() => {
                setShowCustom(!showCustom)
                setValue(FieldName.PROVIDER_SOURCE, "", {
                  shouldDirty: true,
                })
              }}
            />
          </SimpleGrid>
          {showCustom && (
            <Box marginTop={"10px"}>
              <FormLabel>Source</FormLabel>
              <Input
                placeholder="Enter provider source"
                type="text"
                {...register(FieldName.PROVIDER_SOURCE, { required: true })}
              />
              {providerSourceError && providerSourceError.message ? (
                <FormErrorMessage>{providerSourceError.message ?? "Error"}</FormErrorMessage>
              ) : (
                <FormHelperText>
                  Can either be a URL or local path to a <Code>provider</Code> binary, or a github
                  repo in the form of <Code>$ORG/$REPO</Code>, i.e.{" "}
                  <Code>loft-sh/devpod-provider-loft</Code>
                </FormHelperText>
              )}
            </Box>
          )}
        </FormControl>
        <CollapsibleSection title={"Advanced Options"} showIcon={true}>
          <FormControl
            isDisabled={!exists(suggestedProviderName)}
            isInvalid={exists(providerNameError)}>
            <FormLabel>Custom Name</FormLabel>
            <Input
              placeholder="Custom provider name"
              type="text"
              {...register(FieldName.PROVIDER_NAME, {
                pattern: {
                  value: ALLOWED_NAMES_REGEX,
                  message: "Name can only contain letters, numbers, . and -",
                },
              })}
            />
            {exists(providerNameError) ? (
              <FormErrorMessage>{providerNameError.message ?? "Error"}</FormErrorMessage>
            ) : (
              <FormHelperText>
                Optionally give your provider a different name from the one specified in its{" "}
                <Code>provider.yaml</Code>
              </FormHelperText>
            )}
          </FormControl>
        </CollapsibleSection>
        )
        <VStack align="start">
          {status === "error" && isError(error) && <ErrorMessageBox error={error} />}
          <Button type="submit" isDisabled={isSubmitDisabled} isLoading={status === "loading"}>
            Continue
          </Button>
        </VStack>
      </Stack>
    </Form>
  )
}
