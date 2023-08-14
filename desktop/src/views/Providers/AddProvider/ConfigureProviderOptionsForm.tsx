import { BottomActionBar, BottomActionBarError, CollapsibleSection } from "@/components"
import {
  Box,
  Button,
  Checkbox,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  HStack,
  SimpleGrid,
  Spinner,
  Tooltip,
  VStack,
  useColorModeValue,
} from "@chakra-ui/react"
import styled from "@emotion/styled"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { RefObject, useCallback, useEffect, useMemo, useRef } from "react"
import { DefaultValues, FormProvider, SubmitHandler, useForm } from "react-hook-form"
import { useBorderColor } from "../../../Theme"
import { client } from "../../../client"
import { useProvider } from "../../../contexts"
import { exists, useFormErrors } from "../../../lib"
import { QueryKeys } from "../../../queryKeys"
import { TConfigureProviderConfig, TProviderID, TProviderOptions } from "../../../types"
import { TOptionWithID, canCreateMachine, getVisibleOptions } from "../helpers"
import { OptionFormField } from "./OptionFormField"

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
  [key: string]: string | boolean | undefined
}>
type TConfigureProviderOptionsFormProps = Readonly<{
  providerID: TProviderID
  isModal?: boolean
  addProvider?: boolean
  isDefault: boolean
  reuseMachine: boolean
  containerRef?: RefObject<HTMLDivElement>
  showBottomActionBar?: boolean
  onFinish?: () => void
}>

export function ConfigureProviderOptionsForm({
  containerRef,
  providerID,
  onFinish,
  isDefault,
  reuseMachine,
  addProvider = false,
  isModal = false,
  showBottomActionBar = true,
}: TConfigureProviderOptionsFormProps) {
  const queryClient = useQueryClient()
  const [provider] = useProvider(providerID)
  const { data: queryOptions, error: queryError } = useQuery<TProviderOptions | undefined, Error>({
    queryKey: QueryKeys.providerSetOptions(providerID!),
    queryFn: async () =>
      (await client.providers.setOptionsDry(providerID!, { options: {} })).unwrap(),
    enabled: true,
  })
  const optionGroups = useMemo(
    () => provider?.config?.optionGroups ?? [],
    [provider?.config?.optionGroups]
  )

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
    error: configureError,
    mutate: configureProvider,
  } = useMutation<
    void,
    Error,
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

  const {
    data: refreshOptions,
    error: refreshError,
    status: refreshStatus,
    mutate: refreshSubOptionsMutation,
  } = useMutation<
    TProviderOptions | undefined,
    Error,
    Readonly<{ providerID: TProviderID; config: TConfigureProviderConfig }>
  >({
    mutationFn: async ({ providerID, config }) => {
      return (await client.providers.setOptionsDry(providerID, config)).unwrap()
    },
    onSuccess(data) {
      if (!data) {
        return
      }

      const newOptions: DefaultValues<TFieldValues> = {}
      Object.keys(data).forEach((key) => {
        if (data[key]?.value) {
          newOptions[key] = data[key]?.value ?? undefined
        }
      })

      formMethods.reset(newOptions)
    },
  })

  const error = useMemo(() => {
    if (configureError) {
      return configureError
    } else if (queryError) {
      return queryError
    } else if (refreshError) {
      return refreshError
    }

    return undefined
  }, [queryError, configureError, refreshError])

  // Open error popover when error changes
  const errorButtonRef = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    if (error) {
      errorButtonRef.current?.click()
    }
  }, [error])

  const optionsProp = useMemo(() => {
    if (refreshOptions) {
      return refreshOptions
    } else if (queryOptions) {
      return queryOptions
    }

    return undefined
  }, [queryOptions, refreshOptions])
  const onSubmit = useCallback<SubmitHandler<TFieldValues>>(
    (data) => {
      const { useAsDefault, reuseMachine } = data
      configureProvider({
        providerID,
        config: {
          reuseMachine: reuseMachine ?? false,
          useAsDefaultProvider: useAsDefault,
          options: filterOptions(data, optionsProp),
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
      const optionGroup = optionGroups.find((group) => {
        return group.options?.find((o) => {
          return optionMatches(o, option.id)
        })
      })
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

  const refreshSubOptions = useCallback(
    (id: string) => {
      const filteredOptions = filterOptions(formMethods.getValues(), optionsProp)
      stripOptionChildren(filteredOptions, optionsProp, id)

      refreshSubOptionsMutation({
        providerID,
        config: {
          options: filteredOptions,
        },
      })
    },
    [formMethods, optionsProp, providerID, refreshSubOptionsMutation]
  )

  if (!exists(provider) || !optionsProp) {
    return <Spinner style={{ margin: "0 auto 3rem auto" }} />
  }

  return (
    <FormProvider {...formMethods}>
      {refreshStatus === "loading" && (
        <div
          style={{
            position: "absolute",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            top: "0",
            left: "0",
            right: "0",
            bottom: "0",
            zIndex: "99999999",
            backgroundColor: "rgba(0,0,0,0.5)",
          }}>
          <Spinner style={{ margin: "auto" }} />
        </div>
      )}
      <Form aria-readonly={true} onSubmit={formMethods.handleSubmit(onSubmit)}>
        <VStack align="start" spacing={8}>
          {options.required.length > 0 && (
            <Box width="full">
              <VStack align="start" spacing={4}>
                {options.required.map((option) => (
                  <OptionFormField
                    key={option.id}
                    refreshSubOptions={refreshSubOptions}
                    isRequired
                    {...option}
                  />
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
                      {group.options?.reduce((arr: Array<JSX.Element>, optionName) => {
                        groupOptions?.forEach((option) => {
                          if (optionMatches(optionName, option.id)) {
                            arr.push(
                              <OptionFormField
                                key={option.id}
                                refreshSubOptions={refreshSubOptions}
                                isRequired={!!option.required}
                                {...option}
                              />
                            )
                          }
                        })

                        return arr
                      }, [] as Array<JSX.Element>)}
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
                    <OptionFormField
                      key={option.id}
                      refreshSubOptions={refreshSubOptions}
                      {...option}
                    />
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

          {showBottomActionBar && (
            <BottomActionBar isModal={isModal}>
              <HStack>
                <Tooltip
                  label="Please configure provider"
                  isDisabled={formMethods.formState.isValid}>
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

              <BottomActionBarError error={error} containerRef={containerRef} />
            </BottomActionBar>
          )}
        </VStack>
      </Form>
    </FormProvider>
  )
}

function optionMatches(optionName: string, optionID: string): boolean {
  if (optionName.includes("*")) {
    const regEx = new RegExp("^" + optionName.replaceAll("*", ".*") + "$")

    return regEx.test(optionID)
  }

  return optionName === optionID
}

function stripOptionChildren(
  configuredOptions: { [key: string]: string },
  optionsProp: TProviderOptions | undefined,
  id: string
) {
  // filter children
  optionsProp?.[id]?.children?.forEach((child) => {
    delete configuredOptions[child]
    stripOptionChildren(configuredOptions, optionsProp, child)
  })
}

function filterOptions(
  configuredOptions: TFieldValues,
  optionsProp: TProviderOptions | undefined
): { [key: string]: string } {
  const newOptions: { [key: string]: string } = {}
  Object.keys(configuredOptions).forEach((option) => {
    if (exists(configuredOptions[option]) && exists(optionsProp?.[option])) {
      newOptions[option] = configuredOptions[option] + ""
    }
  })

  return newOptions
}
