import {
  Button,
  Code,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Input,
  Stack,
  VStack,
} from "@chakra-ui/react"
import styled from "@emotion/styled"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useCallback, useDeferredValue, useEffect, useMemo } from "react"
import { SubmitHandler, useForm } from "react-hook-form"
import { client } from "../../../client"
import { CollapsibleSection, ErrorMessageBox } from "../../../components"
import { exists, isEmpty, isError, useFormErrors } from "../../../lib"
import { QueryKeys } from "../../../queryKeys"
import {
  TAddProviderConfig,
  TProviderOptionGroup,
  TProviderOptions,
  TWithProviderID,
} from "../../../types"

const Form = styled.form`
  width: 100%;
`
const FieldName = {
  PROVIDER_SOURCE: "providerSource",
  PROVIDER_NAME: "providerName",
} as const
const ALLOWED_NAMES_REGEX = /^[a-z0-9\\.\\-]+$/
type TFormValues = {
  [FieldName.PROVIDER_SOURCE]: string
  [FieldName.PROVIDER_NAME]: string | undefined
}

type TSetupProviderSourceFormProps = Readonly<{
  onFinish: (
    result: TWithProviderID &
      Readonly<{ options: TProviderOptions; optionGroups: TProviderOptionGroup[] }>
  ) => void
}>
export function SetupProviderSourceForm({ onFinish }: TSetupProviderSourceFormProps) {
  const { register, handleSubmit, formState, watch, setValue } = useForm<TFormValues>({
    mode: "onBlur",
  })
  const providerSource = watch(FieldName.PROVIDER_SOURCE, "")
  const deferredProviderSource = useDeferredValue(providerSource)

  const queryClient = useQueryClient()
  const { data: suggestedProviderName } = useQuery({
    queryKey: ["providerNameSuggestion", deferredProviderSource],
    queryFn: async () => {
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

  return (
    <Form onSubmit={handleSubmit(onSubmit)} spellCheck={false}>
      <Stack spacing={6} width="full">
        <FormControl isRequired isInvalid={exists(providerSourceError)}>
          <FormLabel>Source</FormLabel>
          <Input
            placeholder="Enter provider source"
            type="text"
            {...register(FieldName.PROVIDER_SOURCE, { required: true })}
          />
          {exists(providerSourceError) ? (
            <FormErrorMessage>{providerSourceError.message ?? "Error"}</FormErrorMessage>
          ) : (
            <FormHelperText>
              Can either be a URL or local path to a <Code>provider</Code> binary, or a github repo
              in the form of <Code>$ORG/$REPO</Code>, i.e. <Code>loft-sh/devpod-provider-loft</Code>
            </FormHelperText>
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
