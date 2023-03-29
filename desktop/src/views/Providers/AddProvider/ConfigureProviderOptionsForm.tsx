import {
  Box,
  Button,
  Checkbox,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Heading,
  Input,
  SimpleGrid,
  VStack,
} from "@chakra-ui/react"
import styled from "@emotion/styled"
import { useMutation } from "@tanstack/react-query"
import { ReactNode, useCallback, useMemo } from "react"
import { FormProvider, SubmitHandler, useForm, useFormContext } from "react-hook-form"
import { client } from "../../../client"
import { useProvider } from "../../../contexts"
import { exists, useFormErrors } from "../../../lib"
import { TConfigureProviderConfig, TProviderID, TProviderOptions } from "../../../types"
import { canCreateMachine, getVisibleOptions, TOptionWithID } from "../helpers"

type TAllOptions = Readonly<{ required: TOptionWithID[]; other: TOptionWithID[] }>
const Form = styled.form`
  width: 100%;
`

const FieldName = {
  REUSE_MACHINE: "reuseMachine",
  USE_AS_DEFAULT: "useAsDefault",
} as const

type TFieldValues = Readonly<{
  [FieldName.REUSE_MACHINE]: boolean | undefined
  [FieldName.USE_AS_DEFAULT]: boolean
  [key: string]: unknown
}>
type TConfigureProviderOptionsFormProps = Readonly<{
  providerID: TProviderID
  options: TProviderOptions
  initializeProvider?: boolean
  onFinish?: () => void
}>
export function ConfigureProviderOptionsForm({
  providerID,
  onFinish,
  options: optionsProp,
  initializeProvider = false,
}: TConfigureProviderOptionsFormProps) {
  const [[provider]] = useProvider(providerID)
  const formMethods = useForm<TFieldValues>()
  const { status, mutate: configureProvider } = useMutation({
    mutationFn: async ({
      providerID,
      config,
    }: Readonly<{ providerID: TProviderID; config: TConfigureProviderConfig }>) =>
      client.providers.configure(providerID, config),
    onSuccess() {
      onFinish?.()
    },
  })

  const onSubmit = useCallback<SubmitHandler<TFieldValues>>(
    (data) => {
      const { useAsDefault, reuseMachine, ...options } = data

      configureProvider({
        providerID,
        config: {
          initializeProvider,
          reuseMachine: reuseMachine ?? false,
          useAsDefaultProvider: useAsDefault,
          options: options,
        },
      })
    },
    [configureProvider, initializeProvider, providerID]
  )
  const { reuseMachineError, useAsDefaultError } = useFormErrors(
    Object.values(FieldName),
    formMethods.formState
  )
  const showReuseMachineField = useMemo(
    () => canCreateMachine(provider?.config),
    [provider?.config]
  )
  const showUseAsDefaultField = useMemo(() => initializeProvider, [initializeProvider])

  const options = useMemo(() => {
    const empty: TAllOptions = { required: [], other: [] }
    if (!exists(optionsProp)) {
      return empty
    }

    return getVisibleOptions(optionsProp).reduce<TAllOptions>((acc, option) => {
      if (option.required) {
        acc.required.push(option)

        return acc
      }

      acc.other.push(option)

      return acc
    }, empty)
  }, [optionsProp])

  return (
    <FormProvider {...formMethods}>
      <Form onSubmit={formMethods.handleSubmit(onSubmit)}>
        <VStack align="start" spacing={14}>
          {options.required.length > 0 && (
            <Box width="full">
              <Heading size="sm" marginBottom={4}>
                Required
              </Heading>
              <VStack align="start" spacing={4}>
                {options.required.map((option) => (
                  <OptionFormField key={option.id} isRequired {...option} />
                ))}
              </VStack>
            </Box>
          )}

          {options.other.length > 0 && (
            <Box width="full">
              <Heading size="sm" marginBottom={4}>
                Optional
              </Heading>
              <SimpleGrid minChildWidth="60" spacingX={8} spacingY={4}>
                {options.other.map((option) => (
                  <OptionFormField key={option.id} {...option} />
                ))}
              </SimpleGrid>
            </Box>
          )}

          {(showReuseMachineField || showUseAsDefaultField) && (
            <Box width="full">
              <Heading size="sm" marginBottom={4}>
                Other Options
              </Heading>
              <VStack align="start" spacing={4}>
                {showReuseMachineField && (
                  <FormControl>
                    <Checkbox {...formMethods.register(FieldName.REUSE_MACHINE)}>
                      Reuse Machine
                    </Checkbox>
                    {exists(reuseMachineError) ? (
                      <FormErrorMessage>{reuseMachineError.message ?? "Error"}</FormErrorMessage>
                    ) : (
                      <FormHelperText>
                        Provider reuses the vm of the first workspaces for all subsequent
                        workspaces. Otherwise, it will spin up one VM per workspace
                      </FormHelperText>
                    )}
                  </FormControl>
                )}

                {showUseAsDefaultField && (
                  <FormControl>
                    <Checkbox {...formMethods.register(FieldName.USE_AS_DEFAULT)}>
                      Default Provider
                    </Checkbox>
                    {exists(useAsDefaultError) ? (
                      <FormErrorMessage>{useAsDefaultError.message ?? "Error"}</FormErrorMessage>
                    ) : (
                      <FormHelperText>Use this provider as the default provider</FormHelperText>
                    )}
                  </FormControl>
                )}
              </VStack>
            </Box>
          )}

          <Button
            marginTop="10"
            type="submit"
            isLoading={status === "loading"}
            disabled={formMethods.formState.isSubmitting}>
            {initializeProvider ? "Create Provider" : "Save"}
          </Button>
        </VStack>
      </Form>
    </FormProvider>
  )
}

type TOptionFormField = TOptionWithID & Readonly<{ isRequired?: boolean }>
function OptionFormField({
  id,
  defaultValue,
  value,
  description,
  type,
  displayName,
  isRequired = false,
}: TOptionFormField) {
  const { register, formState } = useFormContext()
  const optionError = formState.errors[id]

  const input = useMemo<ReactNode>(() => {
    const registerProps = register(id, { required: isRequired })
    const valueProp = exists(value) ? { defaultValue: value } : {}
    const defaultValueProp = exists(defaultValue) ? { defaultValue } : {}
    const props = { ...defaultValueProp, ...valueProp, ...registerProps }

    switch (type) {
      case "boolean":
        return <Checkbox {...props}>{displayName}</Checkbox>
      case "number":
        return <Input placeholder={`Enter ${displayName}`} type="number" {...props} />
      case "duration":
        return <Input placeholder={`Enter ${displayName}`} type="text" {...props} />
      case "string":
        return <Input placeholder={`Enter ${displayName}`} type="text" {...props} />
      default:
        return <Input placeholder={`Enter ${displayName}`} type="text" {...props} />
    }
  }, [defaultValue, displayName, id, isRequired, register, type, value])

  return (
    <FormControl isRequired={isRequired}>
      <FormLabel>{displayName}</FormLabel>
      {input}
      {exists(optionError) ? (
        <FormErrorMessage>{optionError.message?.toString() ?? "Error"}</FormErrorMessage>
      ) : (
        exists(description) && <FormHelperText>{description}</FormHelperText>
      )}
    </FormControl>
  )
}
