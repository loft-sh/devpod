import {
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
import { useCallback, useEffect, useMemo, useState } from "react"
import { SubmitHandler, useForm } from "react-hook-form"
import { useNavigate } from "react-router"
import { CollapsibleSection, useStreamingTerminal } from "../../components"
import { useProviders, useWorkspace, useWorkspaces } from "../../contexts"
import { exists, useFormErrors } from "../../lib"
import { Routes } from "../../routes"
import { TProviderID } from "../../types"
import { ExampleCard } from "./ExampleCard"
import GolangPng from "../../images/go.png"
import NodeJSPng from "../../images/nodejs.png"
import { client } from "../../client"
import { FieldName, SUPPORTED_IDES, TFormValues } from "./types"

const DEFAULT_PROVIDER = "docker"

// TODO: handle no provider configured
export function CreateWorkspace() {
  const navigate = useNavigate()
  const workspaces = useWorkspaces()
  const [workspaceID, setWorkspaceID] = useState<string | undefined>(undefined)
  const workspace = useWorkspace(workspaceID)
  const [[providers]] = useProviders()
  const { register, handleSubmit, formState, watch, setError, setValue } = useForm<TFormValues>({
    defaultValues: {
      [FieldName.DEFAULT_IDE]: "vscode",
      [FieldName.PROVIDER]: DEFAULT_PROVIDER,
    },
  })
  const currentSource = watch(FieldName.SOURCE)
  const { terminal, connectStream } = useStreamingTerminal()

  const onSubmit = useCallback<SubmitHandler<TFormValues>>(
    async (data) => {
      const workspaceSource = data[FieldName.SOURCE].trim()
      const newIDResult = await client.workspaces.newID(workspaceSource)
      if (newIDResult.err) {
        setError(FieldName.SOURCE, { message: newIDResult.val.message })

        return
      } else if (workspaces.find((workspace) => workspace.id === newIDResult.val)) {
        setError(FieldName.SOURCE, { message: "workspace with the same name already exists" })

        return
      }

      const workspaceID = newIDResult.val
      const providerID = data[FieldName.PROVIDER]
      const defaultIDE = data[FieldName.DEFAULT_IDE]

      // TODO: after creating a workspace, the status is NOT_FOUND until the whole devcontainer is set up...
      // can we change this in cli?
      workspace.create(
        {
          id: workspaceID,
          providerConfig: { providerID },
          ideConfig: { ide: defaultIDE },
          sourceConfig: {
            source: workspaceSource,
          },
        },
        connectStream
      )

      // set workspace id to show terminal
      setWorkspaceID(workspaceID)
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
              <FormControl isRequired isInvalid={exists(sourceError)}>
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
                {exists(sourceError) && (
                  <FormErrorMessage>{sourceError.message ?? "Error"}</FormErrorMessage>
                )}
              </FormControl>
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
