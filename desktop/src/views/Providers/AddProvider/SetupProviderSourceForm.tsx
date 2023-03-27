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
import { useMutation, useQuery } from "@tanstack/react-query"
import { useCallback, useDeferredValue, useEffect, useMemo } from "react"
import { SubmitHandler, useForm } from "react-hook-form"
import { client } from "../../../client"
import { ErrorMessageBox } from "../../../components"
import { exists, isEmpty, isError, useFormErrors } from "../../../lib"
import { TAddProviderConfig, TProviderOptions, TWithProviderID } from "../../../types"

const Form = styled.form`
  width: 100%;
`
const FieldName = {
  PROVIDER_SOURCE: "providerSource",
  PROVIDER_NAME: "providerName",
} as const
const ALLOWED_NAMES_REGEX = /^[a-zA-Z0-9\\.\\-]+$/
type TFormValues = {
  [FieldName.PROVIDER_SOURCE]: string
  [FieldName.PROVIDER_NAME]: string | undefined
}

type TSetupProviderSourceFormProps = Readonly<{
  onFinish: (result: TWithProviderID & Readonly<{ options: TProviderOptions }>) => void
}>
export function SetupProviderSourceForm({ onFinish }: TSetupProviderSourceFormProps) {
  const { register, handleSubmit, formState, watch, setValue } = useForm<TFormValues>({
    mode: "onBlur",
  })
  const providerSource = watch(FieldName.PROVIDER_SOURCE, "")
  const deferredProviderSource = useDeferredValue(providerSource)

  const { data: suggestedProviderName } = useQuery({
    queryKey: ["providerNameSuggestion", deferredProviderSource],
    queryFn: () => {
      return client.providers.newID(deferredProviderSource)
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
      await client.providers.add(rawProviderSource, config)
      const providerID = await client.providers.newID(rawProviderSource)
      const options = await client.providers.getOptions(providerID)

      return { providerID, options }
    },
    onSuccess(result) {
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
    const watchProviderSource = watch((_, { name }) => {
      if (name !== FieldName.PROVIDER_SOURCE) {
        return
      }

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
    return status === "error" || !formState.dirtyFields[FieldName.PROVIDER_SOURCE]
  }, [formState.dirtyFields, status])

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
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
        )
        <VStack align="start">
          {status === "error" && isError(error) && <ErrorMessageBox error={error} />}
          <Button
            marginTop="10"
            type="submit"
            isDisabled={isSubmitDisabled}
            isLoading={status === "loading"}
            disabled={formState.isSubmitting}>
            Continue
          </Button>
        </VStack>
      </Stack>
    </Form>
  )
}
