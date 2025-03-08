import { BottomActionBar, BottomActionBarError, CollapsibleSection } from "@/components"
import { CheckIcon } from "@chakra-ui/icons"
import {
  Box,
  Button,
  Center,
  Checkbox,
  CircularProgress,
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
import {
  FormEventHandler,
  RefObject,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react"
import { DefaultValues, FormProvider, UseFormReturn, useForm } from "react-hook-form"
import { useBorderColor } from "../../../Theme"
import { client } from "../../../client"
import { useProvider } from "../../../contexts"
import { exists, useFormErrors } from "../../../lib"
import { QueryKeys } from "../../../queryKeys"
import {
  TConfigureProviderConfig,
  TProvider,
  TProviderID,
  TProviderOption,
  TProviderOptions,
  TWorkspace,
} from "../../../types"
import { canCreateMachine } from "../helpers"
import { OptionFormField } from "./OptionFormField"
import { useProviderDisplayOptions } from "./useProviderOptions"

const Form = styled.form`
  width: 100%;
  position: relative;
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
type TCommonProps = Readonly<{
  providerID: TProviderID
  isModal?: boolean
  addProvider?: boolean
  isDefault?: boolean
  reuseMachine: boolean
  containerRef?: RefObject<HTMLDivElement>
  showBottomActionBar?: boolean
  suggestedOptions?: Record<string, string>
  submitTitle?: string
}>
type TConfigureProviderOptionsFormProps = TWithWorkspace | TWithoutWorkspace
type TWithWorkspace = Readonly<{
  workspace: TWorkspace
  onFinish?: (extraProviderOptions: Record<string, string>) => void
}> &
  TCommonProps
type TWithoutWorkspace = Readonly<{
  workspace?: never
  onFinish?: () => void
}> &
  TCommonProps

export function ConfigureProviderOptionsForm(props: TConfigureProviderOptionsFormProps) {
  if (props.workspace !== undefined) {
    // configure provider options on workspace level
    return <WorkspaceProviderOptionsForm {...props} workspace={props.workspace} />
  }

  // configure provider
  return <ProviderOptionsForm {...props} />
}

type TProviderOptionsFormProps = TWithoutWorkspace
function ProviderOptionsForm(props: TProviderOptionsFormProps) {
  const queryClient = useQueryClient()
  const handleSave: TConfigureOptionsFormProps["onSave"] = useCallback(
    async ({ providerID, config }) => {
      ;(await client.providers.configure(providerID, config)).unwrap()
      await queryClient.invalidateQueries(QueryKeys.PROVIDERS)
    },
    [queryClient]
  )

  return <ConfigureOptionsForm {...props} onSave={handleSave} />
}

type TWorkspaceProviderOptionsFormProps = TWithWorkspace
function WorkspaceProviderOptionsForm({ workspace, ...props }: TWorkspaceProviderOptionsFormProps) {
  const handleSave: TConfigureOptionsFormProps["onSave"] = useCallback(async () => {
    /* noop */
  }, [])

  return <ConfigureOptionsForm {...props} workspace={workspace} onSave={handleSave} />
}

type TConfigureOptionsFormProps = TConfigureProviderOptionsFormProps &
  Readonly<{
    onSave: (
      args: Readonly<{ providerID: TProviderID; config: TConfigureProviderConfig }>
    ) => Promise<void>
  }>
function ConfigureOptionsForm({
  containerRef,
  providerID,
  onFinish,
  isDefault,
  reuseMachine,
  addProvider = false,
  isModal = false,
  showBottomActionBar = true,
  submitTitle,
  suggestedOptions,
  workspace,
  onSave,
}: TConfigureOptionsFormProps) {
  const loadingBgColor = useColorModeValue("whiteAlpha.400", "blackAlpha.400")
  const [provider] = useProvider(providerID)

  const formMethods = useForm<TFieldValues>({
    defaultValues: {
      reuseMachine,
      useAsDefault: isDefault,
    },
  })
  const { reuseMachineError, useAsDefaultError } = useFormErrors(
    Object.values(FieldName),
    formMethods.formState
  )

  const {
    allOptions,
    displayOptions,
    error: optionsError,
    refresh: refreshOptions,
    isRefreshing,
  } = useOptions(providerID, provider, workspace, suggestedOptions, formMethods)

  const {
    status,
    error: configureError,
    mutate: configureProvider,
  } = useMutation<
    void,
    Error,
    Readonly<{ providerID: TProviderID; config: TConfigureProviderConfig }>
  >({
    mutationFn: onSave,
    onSuccess(_, { config: { options } }) {
      formMethods.reset(
        { reuseMachine, useAsDefault: isDefault },
        { keepValues: true, keepDirty: false }
      )
      onFinish?.(options)
    },
  })

  const error = useMemo(
    () => configureError ?? optionsError ?? undefined,
    [configureError, optionsError]
  )
  // Open error popover when error changes
  const errorButtonRef = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    if (error) {
      errorButtonRef.current?.click()
    }
  }, [error])

  const backgroundColor = useColorModeValue("gray.50", "gray.800")
  const borderColor = useBorderColor()

  const handleRefreshSubOptions = useCallback(
    (id: string) => refreshOptions({ targetOptionID: id }),
    [refreshOptions]
  )

  const handleSubmit: FormEventHandler<HTMLFormElement> = (event) => {
    // make sure we don't bubble up the event to the parent
    event.stopPropagation()
    event.preventDefault()

    formMethods.handleSubmit((data) => {
      const { useAsDefault, reuseMachine } = data
      configureProvider({
        providerID,
        config: {
          reuseMachine: reuseMachine ?? false,
          useAsDefaultProvider: useAsDefault,
          options: filterOptions(data, allOptions),
        },
      })
    })(event)
  }

  const submitButtonText = useMemo(() => {
    if (submitTitle !== undefined) {
      return submitTitle
    }

    return addProvider ? "Add Provider" : "Update Options"
  }, [addProvider, submitTitle])

  const showDefaultField = useMemo(() => {
    if (isDefault === undefined) {
      return false
    }

    return addProvider || !isDefault
  }, [addProvider, isDefault])

  const showReuseMachineField = useMemo(
    () => canCreateMachine(provider?.config),
    [provider?.config]
  )

  if (!exists(provider) || !allOptions) {
    return <Spinner style={{ margin: "0 auto 3rem auto" }} />
  }

  return (
    <FormProvider {...formMethods}>
      <Form aria-readonly={true} onSubmit={handleSubmit}>
        <VStack align="start" width="full">
          <VStack align="start" spacing={8} position="relative" width="full">
            <Center
              opacity={isRefreshing ? "1" : "0"}
              pointerEvents={isRefreshing ? "auto" : "none"}
              transitionDuration="150ms"
              transitionProperty="opacity,backdrop-filter"
              position="absolute"
              zIndex="tooltip"
              top="0"
              bottom="0"
              width="full"
              backdropFilter="auto"
              backdropBlur="2px"
              bgColor={loadingBgColor}>
              <CircularProgress isIndeterminate boxSize={6} margin="auto" color="gray.700" />
            </Center>

            {displayOptions.required.length > 0 && (
              <Box width="full">
                <VStack align="start" spacing={4}>
                  {displayOptions.required.map((option) => (
                    <OptionFormField
                      key={option.id}
                      onRefresh={handleRefreshSubOptions}
                      isRequired
                      {...option}
                    />
                  ))}
                </VStack>
              </Box>
            )}

            {displayOptions.groups.map(
              (group) =>
                group.options.length > 0 && (
                  <Box key={group.name} width="full">
                    <CollapsibleSection
                      showIcon={true}
                      title={group.name}
                      isOpen={!!group.defaultVisible}>
                      <SimpleGrid minChildWidth="60" spacingX={8} spacingY={4}>
                        {group.options.map((option) => (
                          <OptionFormField
                            key={option.id}
                            onRefresh={handleRefreshSubOptions}
                            isRequired={!!option.required}
                            {...option}
                          />
                        ))}
                      </SimpleGrid>
                    </CollapsibleSection>
                  </Box>
                )
            )}

            {displayOptions.other.length > 0 && (
              <Box width="full">
                <CollapsibleSection showIcon={true} title={"Optional"} isOpen={false}>
                  <SimpleGrid minChildWidth="60" spacingX={8} spacingY={4}>
                    {displayOptions.other.map((option) => (
                      <OptionFormField
                        key={option.id}
                        onRefresh={handleRefreshSubOptions}
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
                  <FormControl variant="contrast">
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
          </VStack>

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
                    isDisabled={!formMethods.formState.isValid || isRefreshing}
                    rightIcon={
                      status === "success" && !formMethods.formState.isDirty ? (
                        <CheckIcon />
                      ) : undefined
                    }
                    title={submitButtonText}>
                    {submitButtonText}
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

function stripOptionChildren(
  configuredOptions: Record<string, string>,
  allOptions: TProviderOptions | undefined,
  id: string
) {
  // filter children
  allOptions?.[id]?.children?.forEach((child) => {
    delete configuredOptions[child]
    stripOptionChildren(configuredOptions, allOptions, child)
  })
}

function filterOptions(
  configuredOptions: TFieldValues,
  allOptions: TProviderOptions | undefined
): Record<string, string> {
  const newOptions: Record<string, string> = {}
  Object.keys(configuredOptions).forEach((option) => {
    if (exists(configuredOptions[option]) && exists(allOptions?.[option])) {
      newOptions[option] = configuredOptions[option] + ""
    }
  })

  return newOptions
}

function useOptions(
  providerID: string,
  provider: TProvider | undefined,
  workspace: TWorkspace | undefined,
  suggestedOptions: Record<string, string> | undefined,
  formMethods: UseFormReturn<TFieldValues>
) {
  const [isEditingWorkspaceOptions, setIsEditingWorkspaceOptions] = useState(false)
  const { data: queryOptions, error: queryError } = useQuery<TProviderOptions | undefined, Error>({
    queryKey: QueryKeys.providerSetOptions(providerID),
    queryFn: async () => {
      return (
        await client.providers.setOptionsDry(providerID, { options: {}, reconfigure: false })
      ).unwrap()
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
    Readonly<{ targetOptionID?: string; options?: TProviderOptions }>
  >({
    mutationFn: async ({ targetOptionID, options }) => {
      const filteredOptions = filterOptions(formMethods.getValues(), options ?? allOptions)
      if (targetOptionID) {
        stripOptionChildren(filteredOptions, allOptions, targetOptionID)
      }

      return (
        await client.providers.setOptionsDry(providerID, {
          options: filteredOptions,
          reconfigure: true,
        })
      ).unwrap()
    },
    onSuccess(data) {
      if (!data) {
        return
      }

      const newOptions: DefaultValues<TFieldValues> = {}
      for (const option in data) {
        if (data[option]?.value) {
          newOptions[option] = data[option]?.value ?? undefined
        }
      }

      formMethods.reset(newOptions, { keepDirty: true, keepTouched: true, keepSubmitCount: true })
    },
  })

  useEffect(() => {
    if (Object.keys(suggestedOptions ?? {}).length === 0) {
      return
    }

    const opts = suggestedOptions ?? {}
    const changedOptions = []
    for (const option in suggestedOptions) {
      const { isDirty } = formMethods.getFieldState(option)
      if (!isDirty) {
        formMethods.setValue(option, opts[option], {
          shouldDirty: true,
          shouldValidate: true,
          shouldTouch: true,
        })
      }
      changedOptions.push(option)
    }
    if (changedOptions.length > 0) {
      refreshSubOptionsMutation({
        options: changedOptions.reduce((acc, o) => {
          const option = { value: opts[o] } as unknown as TProviderOption

          return { ...acc, [o]: option }
        }, {} as TProviderOptions),
      })
    }
    // only rerun when suggestedOptions changes
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [suggestedOptions])

  useEffect(() => {
    const workspaceOptions = workspace?.provider?.options
    if (!workspaceOptions) {
      return
    }

    const changedOptions: TProviderOptions = {}
    for (const [optionName, option] of Object.entries(workspaceOptions)) {
      const { isDirty } = formMethods.getFieldState(optionName)
      if (!isDirty && option.value !== null) {
        formMethods.setValue(optionName, option.value, {
          shouldDirty: true,
          shouldValidate: true,
          shouldTouch: true,
        })
      }
      changedOptions[optionName] = option
    }
    if (Object.keys(changedOptions).length > 0) {
      setIsEditingWorkspaceOptions(true)
      refreshSubOptionsMutation({ options: changedOptions })
    }
    // only rerun when workspace options change
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workspace?.provider?.options])

  const allOptions = useMemo(() => {
    if (refreshOptions) {
      return refreshOptions
    }

    if (queryOptions) {
      return queryOptions
    }

    return undefined
  }, [queryOptions, refreshOptions])
  const error = queryError ?? refreshError
  const displayOptions = useProviderDisplayOptions(
    allOptions,
    provider?.config?.optionGroups ?? [],
    isEditingWorkspaceOptions
  )
  const isRefreshing = refreshStatus === "loading"

  return { allOptions, displayOptions, error, isRefreshing, refresh: refreshSubOptionsMutation }
}
