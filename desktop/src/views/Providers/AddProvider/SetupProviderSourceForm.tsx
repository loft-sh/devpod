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
  useToken,
  VStack,
} from "@chakra-ui/react"
import styled from "@emotion/styled"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useCallback, useDeferredValue, useEffect, useMemo, useState } from "react"
import { SubmitHandler, useForm } from "react-hook-form"
import { AiOutlinePlusCircle } from "react-icons/ai"
import { client } from "../../../client"
import { CollapsibleSection, ErrorMessageBox } from "../../../components"
import {
  AWSSvg,
  AzureSvg,
  DigitalOceanSvg,
  DockerPng,
  GCloudSvg,
  KubernetesSvg,
} from "../../../images"
import { exists, isEmpty, isError, useFormErrors } from "../../../lib"
import { QueryKeys } from "../../../queryKeys"
import {
  TAddProviderConfig,
  TProviderOptionGroup,
  TProviderOptions,
  TProviders,
  TWithProviderID,
} from "../../../types"
import { RecommendedProviderCard } from "./RecommendedProviderCard"
import { FieldName, TFormValues } from "./types"
import { TSetupProviderState } from "./useSetupProvider"

const Form = styled.form`
  width: 100%;
`
const ALLOWED_NAMES_REGEX = /^[a-z0-9\\-]+$/

const RECOMMENDED_PROVIDER_SOURCES = [
  { image: DockerPng, name: "docker" },
  { image: AWSSvg, name: "aws" },
  { image: GCloudSvg, name: "gcloud" },
  { image: AzureSvg, name: "azure" },
  { image: DigitalOceanSvg, name: "digitalocean" },
  { image: KubernetesSvg, name: "kubernetes" },
] as const

type TSetupProviderSourceFormProps = Readonly<{
  state: TSetupProviderState
  reset: () => void
  onFinish: (
    result: TWithProviderID &
      Readonly<{ options: TProviderOptions; optionGroups: TProviderOptionGroup[] }>
  ) => void
}>
export function SetupProviderSourceForm({ state, reset, onFinish }: TSetupProviderSourceFormProps) {
  const cardSize = useToken("sizes", "36")
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
      // delete the old selected provider
      if (state.currentStep !== 1) {
        const providerID = client.providers.popDangling()
        if (providerID) {
          ;(await client.providers.remove(providerID)).unwrap()
        }
      }

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

  const handleRecommendedProviderClicked = useCallback(
    (sourceName: string) => () => {
      setShowCustom(false)
      setValue(FieldName.PROVIDER_SOURCE, providerSource === sourceName ? "" : sourceName, {
        shouldDirty: true,
      })
      setValue(FieldName.PROVIDER_NAME, providerSource === sourceName ? "" : sourceName, {
        shouldDirty: true,
      })
    },
    [providerSource, setValue]
  )

  return (
    <Form onSubmit={handleSubmit(onSubmit)} spellCheck={false}>
      <Stack spacing={6} width="full">
        <FormControl isRequired isInvalid={exists(providerSourceError)}>
          <SimpleGrid
            spacing={4}
            templateColumns={`repeat(auto-fill, ${cardSize})`}
            marginTop="2.5">
            {RECOMMENDED_PROVIDER_SOURCES.filter(
              (source) => !providers?.[source.name] || !providers[source.name]?.state?.initialized
            ).map((source) => (
              <RecommendedProviderCard
                key={source.name}
                image={source.image}
                source={source.name}
                isSelected={providerSource === source.name}
                onClick={handleRecommendedProviderClicked(source.name)}
              />
            ))}
            <RecommendedProviderCard
              imageNode={<Icon as={AiOutlinePlusCircle} fontSize={"64px"} color={"primary.500"} />}
              isSelected={showCustom}
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
                <FormErrorMessage>{providerSourceError.message}</FormErrorMessage>
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
                  message: "Name can only contain letters, numbers and -",
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
