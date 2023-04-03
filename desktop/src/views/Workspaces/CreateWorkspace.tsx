import {
  Box,
  Button,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  HStack,
  Input,
  Select,
  SimpleGrid,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  VStack,
} from "@chakra-ui/react"
import { useCallback, useEffect, useMemo } from "react"
import { SubmitHandler, useForm } from "react-hook-form"
import { useNavigate } from "react-router"
import { CollapsibleSection, useStreamingTerminal } from "../../components"
import { useProviders, useWorkspace } from "../../contexts"
import { exists, useFormErrors } from "../../lib"
import { Routes } from "../../routes"
import { TProviderID } from "../../types"
import { ExampleCard } from "./ExampleCard"
import GolangPng from "../../images/go.png"
import NodeJSPng from "../../images/nodejs.png"

export const FieldName = {
  SOURCE: "source",
  DEFAULT_IDE: "defaultIDE",
  PROVIDER: "provider",
} as const

const SUPPORTED_IDES = ["vscode", "intellj"] as const
type TSupportedIDE = (typeof SUPPORTED_IDES)[number]

const DEFAULT_PROVIDER = "docker"

export type TFormValues = {
  [FieldName.SOURCE]: string
  [FieldName.DEFAULT_IDE]: TSupportedIDE
  [FieldName.PROVIDER]: TProviderID // TODO: needs runtime validation
}

// TODO: handle no provider configured
export function CreateWorkspace() {
  const navigate = useNavigate()
  const workspace = useWorkspace(undefined)
  const [[providers]] = useProviders()
  const { register, handleSubmit, formState, watch, setValue } = useForm<TFormValues>({
    defaultValues: {
      [FieldName.DEFAULT_IDE]: "vscode",
      [FieldName.PROVIDER]: DEFAULT_PROVIDER,
    },
  })
  const currentSource = watch(FieldName.SOURCE)
  const { terminal, connectStream } = useStreamingTerminal()

  const onSubmit = useCallback<SubmitHandler<TFormValues>>(
    (data) => {
      const workspaceSource = data[FieldName.SOURCE].trim()
      const providerID = data[FieldName.PROVIDER]
      const defaultIDE = data[FieldName.DEFAULT_IDE]

      // TODO: after creating a workspace, the status is NOT_FOUND until the whole devcontainer is set up...
      // can we change this in cli?
      workspace.create(
        {
          providerConfig: { providerID },
          ideConfig: { ide: defaultIDE },
          sourceConfig: {
            source: workspaceSource,
          },
        },
        connectStream
      )
    },
    [workspace, connectStream]
  )

  const { sourceError, providerError, defaultIDEError } = useFormErrors(
    Object.values(FieldName),
    formState
  )

  const providerOptions = useMemo<readonly TProviderID[]>(() => {
    if (!exists(providers)) {
      return [DEFAULT_PROVIDER] // TODO: make dynamic
    }

    return Object.keys(providers)
  }, [providers])

  const isLoading = useMemo(
    () => workspace.current?.name === "create" && workspace.current.status === "pending",
    [workspace.current]
  )

  console.info(workspace)

  useEffect(() => {
    if (workspace.current?.name === "create" && workspace.current.status === "success") {
      navigate(Routes.WORKSPACES)
    }
  }, [navigate, workspace])

  if (isLoading) {
    return terminal
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <VStack align="start" spacing="6">
        <Tabs colorScheme={"primary"} width={"100%"} maxWidth={"1024px"}>
          <TabList>
            <Tab>From Path</Tab>
            <Tab>From Example</Tab>
          </TabList>

          <TabPanels>
            <TabPanel>
              <FormControl isRequired isInvalid={exists(sourceError)}>
                <Input
                  placeholder="github.com/my-org/my-repo"
                  fontSize={"16px"}
                  padding={"10px"}
                  height={"42px"}
                  type="text"
                  {...register(FieldName.SOURCE, { required: true })}
                />
                {exists(sourceError) ? (
                  <FormErrorMessage>{sourceError.message ?? "Error"}</FormErrorMessage>
                ) : (
                  <FormHelperText>
                    Enter any git repository or local path to a folder you would like to create a
                    workspace from.
                  </FormHelperText>
                )}
              </FormControl>
            </TabPanel>
            <TabPanel>
              <SimpleGrid
                spacing={4}
                templateColumns="repeat(auto-fill, minmax(120px, 1fr))"
                marginTop={"10px"}>
                <ExampleCard
                  image={GolangPng}
                  source={"https://github.com/Microsoft/vscode-remote-try-go"}
                  currentSource={currentSource}
                  setValue={setValue}
                />
                <ExampleCard
                  image={NodeJSPng}
                  source={"https://github.com/microsoft/vscode-remote-try-node"}
                  currentSource={currentSource}
                  setValue={setValue}
                />
              </SimpleGrid>
            </TabPanel>
          </TabPanels>
        </Tabs>

        <CollapsibleSection
          title={(isOpen) => (isOpen ? "Hide Advanced Options" : "Show Advanced Options")}>
          <HStack spacing="0">
            <FormControl isRequired isInvalid={exists(defaultIDEError)}>
              <FormLabel>Default IDE</FormLabel>
              <Select
                placeholder="Select Default IDE"
                {...register(FieldName.DEFAULT_IDE, { required: true })}>
                {SUPPORTED_IDES.map((ide) => (
                  <option key={ide} value={ide}>
                    {ide}
                  </option>
                ))}
              </Select>
              {exists(defaultIDEError) ? (
                <FormErrorMessage>{defaultIDEError.message ?? "Error"}</FormErrorMessage>
              ) : (
                <FormHelperText>
                  Devpod will open this workspace with the selected IDE by default. You can still
                  change your default IDE later.
                </FormHelperText>
              )}
            </FormControl>
            <FormControl isRequired isInvalid={exists(providerError)}>
              <FormLabel>Provider</FormLabel>
              <Select
                placeholder="Select Provider"
                {...register(FieldName.PROVIDER, { required: true })}>
                {providerOptions.map((providerID) => (
                  <option key={providerID} value={providerID}>
                    {providerID}
                  </option>
                ))}
              </Select>
              {exists(providerError) ? (
                <FormErrorMessage>{providerError.message ?? "Error"}</FormErrorMessage>
              ) : (
                <FormHelperText>Use this provider to create the workspace.</FormHelperText>
              )}
            </FormControl>
          </HStack>
        </CollapsibleSection>

        <Button
          colorScheme={"primary"}
          marginTop="10"
          type="submit"
          disabled={formState.isSubmitting}>
          Create Workspace
        </Button>
      </VStack>
    </form>
  )
}
