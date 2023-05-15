import {
  Box,
  Button,
  Checkbox,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  HStack,
  IconButton,
  Input,
  Popover,
  PopoverContent,
  PopoverTrigger,
  SimpleGrid,
  Text,
  Tooltip,
  VStack,
  useBreakpointValue,
  useColorModeValue,
} from "@chakra-ui/react"
import styled from "@emotion/styled"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { motion } from "framer-motion"
import { ReactNode, useCallback, useEffect, useMemo, useRef } from "react"
import { Controller, FormProvider, SubmitHandler, useForm, useFormContext } from "react-hook-form"
import { useBorderColor } from "../../../Theme"
import { client } from "../../../client"
import { AutoComplete, CollapsibleSection, ErrorMessageBox } from "../../../components"
import { SIDEBAR_WIDTH } from "../../../constants"
import { useProvider } from "../../../contexts"
import { ExclamationCircle } from "../../../icons"
import { Err, Failed, exists, isError, useFormErrors } from "../../../lib"
import { QueryKeys } from "../../../queryKeys"
import {
  TConfigureProviderConfig,
  TProviderID,
  TProviderOptionGroup,
  TProviderOptions,
} from "../../../types"
import { TOptionWithID, canCreateMachine, getVisibleOptions } from "../helpers"

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
  options: TProviderOptions | undefined
  optionGroups: TProviderOptionGroup[]
  isModal?: boolean
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
  isModal = false,
}: TConfigureProviderOptionsFormProps) {
  const queryClient = useQueryClient()
  const [provider] = useProvider(providerID)
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
  } = useMutation<
    void,
    Err<Failed>,
    Readonly<{ providerID: TProviderID; config: TConfigureProviderConfig }>
  >({
    mutationFn: async ({ providerID, config }) => {
      ;(await client.providers.configure(providerID, config)).unwrap()
      await queryClient.invalidateQueries(QueryKeys.PROVIDERS)
    },
    onSuccess() {
      onFinish?.()
    },
  })

  const errorButtonRef = useRef<HTMLButtonElement>(null)
  // Open error popover when error changes
  useEffect(() => {
    if (error) {
      errorButtonRef.current?.click()
    }
  }, [error])

  const onSubmit = useCallback<SubmitHandler<TFieldValues>>(
    (data) => {
      const { useAsDefault, reuseMachine, ...configuredOptions } = data

      // filter undefined values
      const newOptions: { [key: string]: string } = {}
      Object.keys(configuredOptions).forEach((option) => {
        if (exists(configuredOptions[option]) && exists(optionsProp?.[option])) {
          newOptions[option] = configuredOptions[option] + ""
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
    [configureProvider, optionsProp, providerID]
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
  const translateX = useBreakpointValue({
    base: "translateX(-3rem)",
    xl: isModal ? "translateX(-3rem)" : "",
  })
  const paddingX = useBreakpointValue({ base: "3rem", xl: isModal ? "3rem" : "4" })

  return (
    <FormProvider {...formMethods}>
      <Form onSubmit={formMethods.handleSubmit(onSubmit)}>
        <VStack align="start" spacing={8}>
          {options.required.length > 0 && (
            <Box width="full">
              <VStack align="start" spacing={4}>
                {options.required.map((option) => (
                  <OptionFormField key={option.id} isRequired {...option} />
                ))}
              </VStack>
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

          {showReuseMachineField && (
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
                <FormControl>
                  <Checkbox {...formMethods.register(FieldName.REUSE_MACHINE)}>
                    Reuse Machine
                  </Checkbox>
                  {exists(reuseMachineError) ? (
                    <FormErrorMessage>{reuseMachineError.message ?? "Error"}</FormErrorMessage>
                  ) : (
                    <FormHelperText>
                      Provider will reuse the VM of the first workspace for all subsequent
                      workspaces. Otherwise, it will spin up one VM per workspace.
                    </FormHelperText>
                  )}
                </FormControl>
              </VStack>
            </Box>
          )}

          <HStack
            as={motion.div}
            initial={{ transform: `translateY(100%) ${translateX}` }}
            animate={{ transform: `translateY(0) ${translateX}` }}
            position="sticky"
            bottom="0"
            left="0"
            width={
              isModal
                ? "calc(100% + 5.5rem)"
                : { base: `calc(100vw - ${SIDEBAR_WIDTH})`, xl: "full" }
            }
            height="20"
            backgroundColor="white"
            alignItems="center"
            borderTopWidth="thin"
            borderTopColor={borderColor}
            justifyContent="space-between"
            paddingX={paddingX}
            zIndex="overlay">
            <HStack>
              <Tooltip label="Please configure provider" isDisabled={formMethods.formState.isValid}>
                <Button
                  type="submit"
                  variant="primary"
                  isLoading={formMethods.formState.isSubmitting || status === "loading"}
                  isDisabled={!formMethods.formState.isValid}
                  title={addProvider ? "Add Provider" : "Update Options"}>
                  {addProvider ? "Add Provider" : "Update Options"}
                </Button>
              </Tooltip>

              {showDefaultField && (
                <FormControl
                  paddingX="6"
                  flexDirection="row"
                  display="flex"
                  width="fit-content"
                  isInvalid={exists(useAsDefaultError)}>
                  <Checkbox {...formMethods.register(FieldName.USE_AS_DEFAULT)} />
                  <FormHelperText marginLeft="2" marginTop="0">
                    Set as default{" "}
                  </FormHelperText>
                </FormControl>
              )}
            </HStack>

            <HStack />

            <Popover placement="top" computePositionOnMount>
              <PopoverTrigger>
                <IconButton
                  ref={errorButtonRef}
                  visibility={error ? "visible" : "hidden"}
                  variant="ghost"
                  aria-label="Show errors"
                  icon={
                    <motion.span
                      key={error ? "error" : undefined}
                      animate={{ scale: [1, 1.2, 1] }}
                      transition={{ type: "keyframes", ease: ["easeInOut"] }}>
                      <ExclamationCircle boxSize="8" color="red.400" />
                    </motion.span>
                  }
                  isDisabled={!exists(error)}
                />
              </PopoverTrigger>
              <PopoverContent minWidth="96">
                {isError(error) && <ErrorMessageBox error={error} />}
              </PopoverContent>
            </Popover>
          </HStack>
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
  password,
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
        return (
          <Input spellCheck={false} placeholder={`Enter ${displayName}`} type="number" {...props} />
        )
      case "duration":
        return (
          <Input spellCheck={false} placeholder={`Enter ${displayName}`} type="text" {...props} />
        )
      case "string":
        return (
          <Input
            spellCheck={false}
            placeholder={`Enter ${displayName}`}
            type={password ? "password" : "text"}
            {...props}
          />
        )
      default:
        return (
          <Input
            spellCheck={false}
            placeholder={`Enter ${displayName}`}
            type={password ? "password" : "text"}
            {...props}
          />
        )
    }
  }, [register, id, isRequired, value, defaultValue, suggestions, type, displayName, password])

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
