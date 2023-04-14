import {
  Box,
  Button,
  Checkbox,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Input,
  SimpleGrid,
  useColorModeValue,
  VStack,
} from "@chakra-ui/react"
import styled from "@emotion/styled"
import { useMutation } from "@tanstack/react-query"
import { ReactNode, useCallback, useMemo } from "react"
import { Controller, FormProvider, SubmitHandler, useForm, useFormContext } from "react-hook-form"
import { client } from "../../../client"
import { useProvider } from "../../../contexts"
import { exists, isError, useFormErrors } from "../../../lib"
import {
  TConfigureProviderConfig,
  TProviderID,
  TProviderOptionGroup,
  TProviderOptions,
} from "../../../types"
import { canCreateMachine, getVisibleOptions, TOptionWithID } from "../helpers"
import { AutoComplete, CollapsibleSection, ErrorMessageBox } from "../../../components"
import { useBorderColor } from "../../../Theme"

type TAllOptions = Readonly<{
  required: TOptionWithID[]
  groups: { [key: string]: TOptionWithID[] }
  other: TOptionWithID[]
}>
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
  isDefault: boolean
  reuseMachine: boolean
  options: TProviderOptions
  optionGroups: TProviderOptionGroup[]
  addProvider?: boolean
  onFinish?: () => void
}>

export function ConfigureProviderOptionsForm({
  providerID,
  isDefault,
  reuseMachine,
  onFinish,
  options: optionsProp,
  optionGroups,
  addProvider = false,
}: TConfigureProviderOptionsFormProps) {
  const [[provider]] = useProvider(providerID)
  const showDefaultField = useMemo(() => addProvider || !isDefault, [addProvider, isDefault])
  const showReuseMachineField = useMemo(
    () => canCreateMachine(provider?.config),
    [provider?.config]
  )
  const formMethods = useForm<TFieldValues>({
    defaultValues: {
      useAsDefault: isDefault,
      reuseMachine: reuseMachine,
    },
  })
  const {
    status,
    error,
    mutate: configureProvider,
  } = useMutation({
    mutationFn: async ({
      providerID,
      config,
    }: Readonly<{ providerID: TProviderID; config: TConfigureProviderConfig }>) => {
      ;(await client.providers.configure(providerID, config)).unwrap()
    },
    onSuccess() {
      onFinish?.()
    },
  })

  const onSubmit = useCallback<SubmitHandler<TFieldValues>>(
    (data) => {
      const { useAsDefault, reuseMachine, ...options } = data

      // filter undefined values
      const newOptions: { [key: string]: string } = {}
      Object.keys(options).forEach((option) => {
        if (exists(options[option])) {
          newOptions[option] = options[option] + ""
        }
      })

      configureProvider({
        providerID,
        config: {
          reuseMachine: reuseMachine ?? false,
          useAsDefaultProvider: useAsDefault,
          options: newOptions,
        },
      })
    },
    [configureProvider, providerID]
  )
  const { reuseMachineError, useAsDefaultError } = useFormErrors(
    Object.values(FieldName),
    formMethods.formState
  )

  const options = useMemo(() => {
    const empty: TAllOptions = { required: [], groups: {}, other: [] }
    if (!exists(optionsProp)) {
      return empty
    }

    return getVisibleOptions(optionsProp).reduce<TAllOptions>((acc, option) => {
      const optionGroup = optionGroups.find((group) => group.options?.find((o) => o === option.id))
      if (optionGroup) {
        if (!acc.groups[optionGroup.name!]) {
          acc.groups[optionGroup.name!] = []
        }
        acc.groups[optionGroup.name!]!.push(option)

        return acc
      }

      if (option.required) {
        acc.required.push(option)

        return acc
      }

      acc.other.push(option)

      return acc
    }, empty)
  }, [optionGroups, optionsProp])

  const backgroundColor = useColorModeValue("gray.50", "gray.800")
  const borderColor = useBorderColor()

  return (
    <FormProvider {...formMethods}>
      <Form onSubmit={formMethods.handleSubmit(onSubmit)}>
        <VStack align="start" spacing={8}>
          {options.required.length > 0 && (
            <Box width="full">
              <CollapsibleSection showIcon={true} title={"Required"} isOpen={true}>
                <VStack align="start" spacing={4}>
                  {options.required.map((option) => (
                    <OptionFormField key={option.id} isRequired {...option} />
                  ))}
                </VStack>
              </CollapsibleSection>
            </Box>
          )}

          {optionGroups
            .filter((group) => !!options.groups[group.name!])
            .map((group) => {
              const groupOptions = options.groups[group.name!]

              return (
                <Box key={group.name!} width="full">
                  <CollapsibleSection
                    showIcon={true}
                    title={group.name}
                    isOpen={!!group.defaultVisible}>
                    <SimpleGrid minChildWidth="60" spacingX={8} spacingY={4}>
                      {group.options?.map((optionName) => {
                        const option = groupOptions?.find((option) => option.id === optionName)
                        if (!option) {
                          return undefined
                        }

                        return <OptionFormField key={option.id} {...option} />
                      })}
                    </SimpleGrid>
                  </CollapsibleSection>
                </Box>
              )
            })}

          {options.other.length > 0 && (
            <Box width="full">
              <CollapsibleSection showIcon={true} title={"Optional"} isOpen={false}>
                <SimpleGrid minChildWidth="60" spacingX={8} spacingY={4}>
                  {options.other.map((option) => (
                    <OptionFormField key={option.id} {...option} />
                  ))}
                </SimpleGrid>
              </CollapsibleSection>
            </Box>
          )}

          {(showDefaultField || showReuseMachineField) && (
            <Box width="full">
              <VStack
                align="start"
                spacing={4}
                width="full"
                backgroundColor={backgroundColor}
                borderRadius="lg"
                borderWidth="thin"
                padding={"10px"}
                margin={"10px"}
                borderColor={borderColor}>
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

                {showDefaultField && (
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
          {status === "error" && isError(error) && <ErrorMessageBox error={error} />}
          <Button
            marginTop="10"
            type="submit"
            variant="primary"
            isLoading={status === "loading"}
            disabled={formMethods.formState.isSubmitting}>
            {addProvider ? "Add Provider" : "Update Options"}
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
  suggestions,
  isRequired = false,
}: TOptionFormField) {
  const { register, formState } = useFormContext()
  const optionError = formState.errors[id]

  const input = useMemo<ReactNode>(() => {
    const registerProps = register(id, { required: isRequired })
    const valueProp = exists(value) ? { defaultValue: value } : {}
    const defaultValueProp = exists(defaultValue) ? { defaultValue } : {}
    const props = { ...defaultValueProp, ...valueProp, ...registerProps }

    if (exists(suggestions)) {
      return (
        <Controller
          name={id}
          defaultValue={value ?? defaultValue ?? undefined}
          rules={{ required: isRequired }}
          render={({ field: { onChange, onBlur, value: v, ref } }) => {
            return (
              <AutoComplete
                ref={ref}
                value={v || ""}
                onBlur={onBlur}
                onChange={(value) => {
                  if (value) {
                    onChange(value)
                  }
                }}
                placeholder={`Enter ${displayName}`}
                options={suggestions.map((s) => ({ key: s, label: s }))}
              />
            )
          }}
        />
      )
    }

    switch (type) {
      case "boolean":
        return (
          <Checkbox defaultChecked={props.defaultValue === "true"} {...props}>
            {displayName}
          </Checkbox>
        )
      case "number":
        return <Input placeholder={`Enter ${displayName}`} type="number" {...props} />
      case "duration":
        return <Input placeholder={`Enter ${displayName}`} type="text" {...props} />
      case "string":
        return <Input placeholder={`Enter ${displayName}`} type="text" {...props} />
      default:
        return <Input placeholder={`Enter ${displayName}`} type="text" {...props} />
    }
  }, [register, id, isRequired, value, defaultValue, suggestions, type, displayName])

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
