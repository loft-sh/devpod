import { ExampleCard, Form, IDEIcon, WarningMessageBox } from "@/components"
import {
  Box,
  Button,
  Code,
  Flex,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Grid,
  HStack,
  IconButton,
  Input,
  Link,
  Popover,
  PopoverArrow,
  PopoverBody,
  PopoverCloseButton,
  PopoverContent,
  PopoverHeader,
  PopoverTrigger,
  SimpleGrid,
  Text,
  Tooltip,
  VStack,
  useColorMode,
  useColorModeValue,
  useToken,
} from "@chakra-ui/react"
import { useCallback, useEffect, useMemo, useState } from "react"
import { Controller, ControllerRenderProps } from "react-hook-form"
import { useNavigate } from "react-router"
import { Link as RouterLink } from "react-router-dom"
import { useBorderColor } from "../../../Theme"
import { RECOMMENDED_PROVIDER_SOURCES, SIDEBAR_WIDTH } from "../../../constants"
import { useProvider, useProviders, useWorkspace, useWorkspaces } from "../../../contexts"
import { Plus } from "../../../icons"
import { CommunitySvg, ProviderPlaceholderSvg } from "../../../images"
import { canHealthCheck, exists, getKeys, isEmpty, useFormErrors } from "../../../lib"
import { Routes } from "../../../routes"
import { TIDE, TWorkspace, TWorkspaceSourceType } from "../../../types"
import { useIDEs } from "../../../useIDEs"
import { useSetupProviderModal } from "../../Providers"
import { ProviderOptionsPopover } from "./ProviderOptionsPopover"
import { WorkspaceSourceInput } from "./WorkspaceSourceInput"
import { COMMUNITY_WORKSPACE_EXAMPLES, WORKSPACE_EXAMPLES } from "./constants"
import { FieldName, TCreateWorkspaceArgs, TFormValues, TSelectProviderOptions } from "./types"
import { useCreateWorkspaceForm } from "./useCreateWorkspaceForm"

export function CreateWorkspace() {
  const { ides } = useIDEs()

  const navigate = useNavigate()
  const workspace = useWorkspace<TWorkspace>(undefined)
  const [[providers]] = useProviders()
  const [sourceType, setSourceType] = useState<TWorkspaceSourceType>("git")

  const handleCreateWorkspace = useCallback(
    ({
      workspaceID,
      providerID,
      prebuildRepositories,
      defaultIDE,
      workspaceSource,
      devcontainerPath,
    }: TCreateWorkspaceArgs) => {
      const actionID = workspace.create({
        id: workspaceID,
        prebuildRepositories,
        devcontainerPath,
        providerConfig: { providerID },
        ideConfig: { name: defaultIDE },
        sourceConfig: {
          source: workspaceSource,
          type: sourceType,
        },
      })

      // set action id to show terminal
      if (!isEmpty(actionID)) {
        navigate(Routes.toAction(actionID, Routes.WORKSPACES))
      }
    },
    [navigate, sourceType, workspace]
  )

  const {
    formRef,
    setValue,
    register,
    onSubmit,
    control,
    formState,
    isSubmitting,
    currentSource,
    selectDevcontainerModal,
  } = useCreateWorkspaceForm(handleCreateWorkspace)

  const {
    sourceError,
    providerError,
    defaultIDEError,
    idError,
    prebuildRepositoryError,
    devcontainerPathError,
  } = useFormErrors(Object.values(FieldName), formState)

  const providerOptions = useMemo<TSelectProviderOptions>(() => {
    if (!exists(providers)) {
      return { installed: [], recommended: RECOMMENDED_PROVIDER_SOURCES }
    }

    const installed = Object.entries(providers)
      .filter(([, p]) => !!p.state?.initialized && !canHealthCheck(p.config))
      .map(([key, value]) => ({ name: key, ...value }))

    return {
      installed,
      recommended: RECOMMENDED_PROVIDER_SOURCES,
    }
  }, [providers])

  const handleExampleCardClicked = useCallback(
    (newSource: string) => {
      setValue(FieldName.SOURCE, newSource, {
        shouldDirty: true,
        shouldValidate: true,
      })
    },
    [setValue]
  )

  const {
    isOpen: isModalOpen,
    modal: setupProviderModal,
    show: showSetupProviderModal,
    wasDismissed,
  } = useSetupProviderModal()
  useEffect(() => {
    if (wasDismissed) {
      return
    }

    // no provider available
    if (isEmpty(getKeys(providers || {}))) {
      showSetupProviderModal({ isStrict: true })

      return
    }
  }, [providers, showSetupProviderModal, wasDismissed])

  const backgroundColor = useColorModeValue("gray.50", "gray.900")
  const borderColor = useBorderColor()
  const bottomBarBackgroundColor = useColorModeValue("white", "background.darkest")
  const { colorMode } = useColorMode()

  return (
    <>
      <Form ref={formRef} onSubmit={onSubmit} justifyContent={"center"}>
        <VStack align="start" spacing="6" alignItems="center" width="full" maxWidth="container.lg">
          <HStack
            gap="0"
            width="full"
            height="full"
            borderRadius="lg"
            borderWidth="thin"
            borderColor={borderColor}>
            <FormControl
              borderLeftRadius="lg"
              backgroundColor={backgroundColor}
              paddingX="20"
              paddingY="20"
              height="full"
              isRequired
              isInvalid={exists(sourceError)}
              justifyContent="center"
              display="flex"
              borderRightWidth="thin"
              borderRightColor={borderColor}>
              <VStack width="full">
                <Text width="90%" marginBottom="6" fontWeight="bold">
                  Enter Workspace Source
                </Text>
                <HStack spacing={0} justifyContent={"center"} width="full">
                  <Controller
                    name={FieldName.SOURCE}
                    control={control}
                    rules={{ required: true }}
                    render={({ field }) => (
                      <WorkspaceSourceInput
                        field={field}
                        sourceType={sourceType}
                        onSourceTypeChanged={setSourceType}
                      />
                    )}
                  />
                </HStack>
                {exists(sourceError) && (
                  <FormErrorMessage>{sourceError.message ?? "Error"}</FormErrorMessage>
                )}
                <FormHelperText textAlign={"center"}>
                  Any git repository, local path to a folder or container image you would like to
                  create a workspace from can be a source as long as it adheres to the{" "}
                  <Link
                    fontWeight="bold"
                    target="_blank"
                    href="https://containers.dev/implementors/json_reference/">
                    devcontainer standard
                  </Link>
                  .
                </FormHelperText>
              </VStack>
            </FormControl>

            <Box
              width="72"
              marginInlineStart="0 !important"
              alignSelf="stretch"
              height="full"
              position="relative">
              <Text
                paddingY="3"
                borderBottomWidth="thin"
                borderColor={borderColor}
                width="full"
                marginBottom="4"
                fontWeight="medium"
                textAlign="center">
                Or select one of our quickstart examples
              </Text>
              <FormControl
                paddingTop="2"
                marginBottom="4"
                width="full"
                display="flex"
                flexFlow="column"
                flexWrap="nowrap"
                alignItems="center"
                isRequired
                isInvalid={exists(sourceError)}>
                <SimpleGrid columns={2} spacingX={4} spacingY={4}>
                  {WORKSPACE_EXAMPLES.map((example) => (
                    <ExampleCard
                      key={example.source}
                      size="sm"
                      image={
                        colorMode === "dark" ? example.imageDark ?? example.image : example.image
                      }
                      name={example.name}
                      isSelected={currentSource === example.source}
                      onClick={() => handleExampleCardClicked(example.source)}
                    />
                  ))}
                  <Popover>
                    <PopoverTrigger>
                      <ExampleCard name="Community Quickstart" size="sm" image={CommunitySvg} />
                    </PopoverTrigger>
                    <PopoverContent>
                      <PopoverArrow />
                      <PopoverCloseButton />
                      <PopoverHeader>Community Quickstart</PopoverHeader>
                      <PopoverBody>
                        <Flex gap={4}>
                          {COMMUNITY_WORKSPACE_EXAMPLES.map((example) => (
                            <ExampleCard
                              key={example.source}
                              size="sm"
                              image={colorMode === "dark" ? example.imageDark : example.image}
                              name={example.name}
                              isSelected={currentSource === example.source}
                              onClick={() => handleExampleCardClicked(example.source)}
                            />
                          ))}
                        </Flex>
                      </PopoverBody>
                    </PopoverContent>
                  </Popover>
                </SimpleGrid>
              </FormControl>
            </Box>
          </HStack>

          <VStack spacing="10" maxWidth="container.lg">
            <HStack spacing="8" alignItems={"top"} width={"100%"} justifyContent={"start"}>
              <FormControl isRequired isInvalid={exists(providerError)}>
                <FormLabel>Provider</FormLabel>
                <Controller
                  name={FieldName.PROVIDER}
                  control={control}
                  rules={{ required: true }}
                  render={({ field }) => (
                    <ProviderInput
                      field={field}
                      options={providerOptions}
                      onAddProviderClicked={() =>
                        showSetupProviderModal({
                          isStrict: false,
                        })
                      }
                    />
                  )}
                />
                {exists(providerError) ? (
                  <FormErrorMessage>{providerError.message ?? "Error"}</FormErrorMessage>
                ) : (
                  <FormHelperText>Use this provider to create the workspace.</FormHelperText>
                )}
              </FormControl>

              <FormControl isRequired isInvalid={exists(defaultIDEError)}>
                <FormLabel>Default IDE</FormLabel>
                <Controller
                  name={FieldName.DEFAULT_IDE}
                  control={control}
                  render={({ field }) => (
                    <IDEInput field={field} ides={ides} onClick={(name) => field.onChange(name)} />
                  )}
                />
                {exists(defaultIDEError) ? (
                  <FormErrorMessage>{defaultIDEError.message ?? "Error"}</FormErrorMessage>
                ) : (
                  <FormHelperText>
                    DevPod will open this workspace with the selected IDE by default. You can still
                    change your default IDE later.
                  </FormHelperText>
                )}
              </FormControl>
            </HStack>
            <HStack spacing="8" alignItems={"top"} width={"100%"} justifyContent={"start"}>
              <FormControl isInvalid={exists(idError)}>
                <FormLabel>Workspace Name</FormLabel>
                <Input
                  spellCheck={false}
                  placeholder="my-workspace"
                  type="text"
                  {...register(FieldName.ID, {
                    validate: (value) => {
                      if (/[^a-z0-9-]+/.test(value)) {
                        return "Name can only consist of lower case letters, numbers and dashes"
                      } else {
                        return true
                      }
                    },
                    maxLength: { value: 48, message: "Name cannot be longer than 48 characters" },
                  })}
                  onChange={(e) => {
                    // for some reason this is needed to make `validate` work...
                    setValue(FieldName.ID, e.target.value, {
                      shouldDirty: true,
                      shouldValidate: true,
                    })
                  }}
                />
                {exists(idError) ? (
                  <FormErrorMessage>{idError.message ?? "Error"}</FormErrorMessage>
                ) : (
                  <FormHelperText>
                    This is the workspace name DevPod will use. This is an optional field and
                    usually only needed if you have an already existing workspace with the same
                    name.
                  </FormHelperText>
                )}
              </FormControl>
              <FormControl isInvalid={exists(prebuildRepositoryError)}>
                <FormLabel>Prebuild Repository</FormLabel>
                <Input
                  spellCheck={false}
                  placeholder="ghcr.io/my-org/my-repo"
                  type="text"
                  {...register(FieldName.PREBUILD_REPOSITORY)}
                  onChange={(e) => {
                    setValue(FieldName.PREBUILD_REPOSITORY, e.target.value, {
                      shouldDirty: true,
                      shouldValidate: true,
                    })
                  }}
                />
                {exists(prebuildRepositoryError) ? (
                  <FormErrorMessage>{prebuildRepositoryError.message ?? "Error"}</FormErrorMessage>
                ) : (
                  <FormHelperText>
                    DevPod will use this repository to find prebuilds for the given workspace.
                  </FormHelperText>
                )}
              </FormControl>
            </HStack>

            <HStack spacing="8" alignItems={"top"} width={"100%"} justifyContent={"start"}>
              <FormControl isInvalid={exists(devcontainerPathError)}>
                <FormLabel>Devcontainer Path</FormLabel>
                <Input
                  spellCheck={false}
                  placeholder="Optionally enter path to devcontainer.json"
                  type="text"
                  {...register(FieldName.DEVCONTAINER_PATH, { required: false })}
                />
                {exists(devcontainerPathError) ? (
                  <FormErrorMessage>{devcontainerPathError.message ?? "Error"}</FormErrorMessage>
                ) : (
                  <FormHelperText>
                    DevPod will use this path to create the dev container for this workspace. If not
                    specified it&apos;ll use <Code>.devcontainer.json</Code> or{" "}
                    <Code>.devcontainer/devcontainer.json</Code>. <br />
                    Example: <Code>.devcontainer/service/devcontainer.json</Code>
                  </FormHelperText>
                )}
              </FormControl>
              {/* placholder box */}
              <Box width={"full"} />
            </HStack>
          </VStack>

          <HStack
            position="sticky"
            bottom={{ base: "-1.1rem", xl: "0" }}
            right="0"
            width={{ base: `calc(100vw - ${SIDEBAR_WIDTH})`, xl: "full" }}
            height="20"
            alignItems="center"
            borderTopWidth="thin"
            borderTopColor={borderColor}
            backgroundColor={bottomBarBackgroundColor}
            paddingX={{ base: "8", xl: "0" }}
            paddingY="8"
            zIndex="overlay">
            <Button
              variant="primary"
              type="submit"
              isDisabled={formState.isSubmitting || !formState.isValid}
              isLoading={isSubmitting}>
              Create Workspace
            </Button>
          </HStack>
        </VStack>
      </Form>

      {isModalOpen && setupProviderModal}

      {selectDevcontainerModal}
    </>
  )
}

type TProviderInputProps = Readonly<{
  options: TSelectProviderOptions
  field: ControllerRenderProps<TFormValues, (typeof FieldName)["PROVIDER"]>
  onAddProviderClicked?: VoidFunction
}>
function ProviderInput({ options, field, onAddProviderClicked }: TProviderInputProps) {
  const gridChildWidth = useToken("sizes", "12")
  const [provider] = useProvider(field.value)
  const workspaces = useWorkspaces<TWorkspace>()
  const reuseWorkspace = useMemo(() => {
    return workspaces.find((workspace) => {
      return (
        provider?.state?.singleMachine &&
        workspace.provider?.name === provider.config?.name &&
        workspace.machine?.machineId?.startsWith("devpod-shared-")
      )
    })?.id
  }, [provider, workspaces])

  return (
    <VStack align="start" width="full">
      <Grid
        templateColumns={`repeat(auto-fit, ${gridChildWidth})`}
        gap="2"
        height="fit-content"
        width="full"
        flexWrap="wrap">
        {options.installed.map((p) => (
          <Box key={p.name}>
            <ProviderOptionsPopover
              provider={p}
              trigger={
                <ExampleCard
                  showTooltip={false}
                  isSelected={field.value === p.name}
                  name={p.name}
                  size="sm"
                  onClick={() => field.onChange(p.name)}
                  image={p.config?.icon ?? ProviderPlaceholderSvg}
                />
              }
            />
          </Box>
        ))}
        <Tooltip label="Add Provider">
          <IconButton
            variant="outline"
            size="lg"
            icon={<Plus />}
            aria-label="Add Provider"
            onClick={() => onAddProviderClicked?.()}
          />
        </Tooltip>
      </Grid>

      {reuseWorkspace && (
        <WarningMessageBox
          variant="ghost"
          size="sm"
          warning={
            <span>
              Will reuse the existing machine from {reuseWorkspace} and NOT create a new one. Go to{" "}
              <Link as={RouterLink} to={Routes.toProvider(provider?.config?.name!)}>
                <b>provider settings</b>
              </Link>{" "}
              to change this behaviour.
            </span>
          }
        />
      )}
    </VStack>
  )
}

type TIDEInputProps = Readonly<{
  ides: readonly TIDE[] | undefined
  field: ControllerRenderProps<TFormValues, (typeof FieldName)["DEFAULT_IDE"]>
  onClick: (name: NonNullable<TIDE["name"]>) => void
}>
function IDEInput({ ides, field, onClick }: TIDEInputProps) {
  const gridChildWidth = useToken("sizes", "12")

  return (
    <Grid
      gap={2}
      gridTemplateColumns={{
        lg: `repeat(7, ${gridChildWidth})`,
        xl: `repeat(9, ${gridChildWidth})`,
        "2xl": `repeat(10, ${gridChildWidth})`,
      }}>
      {ides?.map((ide) => {
        const isSelected = field.value === ide.name

        return (
          <Box key={ide.name}>
            <ExampleCard
              name={ide.displayName}
              size="sm"
              image={<IDEIcon ide={ide} />}
              isSelected={isSelected}
              onClick={() => onClick(ide.name!)}
            />
          </Box>
        )
      })}
    </Grid>
  )
}
