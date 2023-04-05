import {
  Button,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  HStack,
  Icon,
  Input,
  InputGroup,
  InputRightElement,
  Select,
  SimpleGrid,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Tooltip,
  VStack,
} from "@chakra-ui/react"
import { open } from "@tauri-apps/api/dialog"
import { useCallback, useEffect, useMemo, useState } from "react"
import { SubmitHandler, useForm } from "react-hook-form"
import { useNavigate } from "react-router"
import { CollapsibleSection, useStreamingTerminal } from "../../components"
import { useProviders, useWorkspace, useWorkspaces } from "../../contexts"
import { exists, useFormErrors } from "../../lib"
import { Routes } from "../../routes"
import { TProviderID, TWorkspaceID } from "../../types"
import { ExampleCard } from "./ExampleCard"
import {
  CppSvg,
  GoPng,
  PhpSvg,
  JavaPng,
  RustSvg,
  NodejsPng,
  PythonSvg,
  DotnetcorePng,
} from "../../images"
import { FiFile } from "react-icons/fi"
import { client } from "../../client"
import { FieldName, TFormValues } from "./types"
import { useQuery } from "@tanstack/react-query"
import { QueryKeys } from "../../queryKeys"
import { useSearchParams } from "react-router-dom"

const DEFAULT_PROVIDER = "docker"

// TODO: handle no provider configured
export function CreateWorkspace() {
  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })

  const params = useCreateWorkspaceParams()
  const [workspaceID, setWorkspaceID] = useState<TWorkspaceID | undefined>(undefined)
  const navigate = useNavigate()
  const workspaces = useWorkspaces()
  const workspace = useWorkspace(workspaceID)
  const [[providers]] = useProviders()
  const { register, handleSubmit, formState, watch, setError, setValue, clearErrors } =
    useForm<TFormValues>({})
  const currentSource = watch(FieldName.SOURCE)
  const { terminal, connectStream } = useStreamingTerminal()

  const onSubmit = useCallback<SubmitHandler<TFormValues>>(
    async (data) => {
      const workspaceSource = data[FieldName.SOURCE].trim()
      let workspaceID = data[FieldName.ID]
      if (!workspaceID) {
        const newIDResult = await client.workspaces.newID(workspaceSource)
        if (newIDResult.err) {
          setError(FieldName.SOURCE, { message: newIDResult.val.message })

          return
        }

        workspaceID = newIDResult.val
      }
      if (workspaces.find((workspace) => workspace.id === workspaceID)) {
        setError(FieldName.SOURCE, { message: "workspace with the same name already exists" })

        return
      }

      const providerID = data[FieldName.PROVIDER]
      const defaultIDE = data[FieldName.DEFAULT_IDE]
      workspace.create(
        {
          id: workspaceID,
          providerConfig: { providerID },
          ideConfig: { name: defaultIDE },
          sourceConfig: {
            source: workspaceSource,
          },
        },
        connectStream
      )

      // set workspace id to show terminal
      setWorkspaceID(workspaceID)
    },
    [workspaces, workspace, connectStream, setError]
  )

  const { sourceError, providerError, defaultIDEError, idError } = useFormErrors(
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
    [workspace]
  )

  useEffect(() => {
    if (
      workspace.current?.name === "create" &&
      workspace.current.status === "success" &&
      workspace.data?.id !== undefined
    ) {
      navigate(Routes.WORKSPACES)
    }
  }, [navigate, workspace])

  if (isLoading) {
    return terminal
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <VStack align="start" spacing="6" marginBottom={"20px"}>
        <Tabs colorScheme={"primary"} width={"100%"} maxWidth={"1024px"}>
          <TabList>
            <Tab>From Path</Tab>
            <Tab>From Example</Tab>
          </TabList>

          <TabPanels>
            <TabPanel>
              <FormControl isRequired isInvalid={exists(sourceError)}>
                <InputGroup>
                  <Input
                    placeholder="github.com/my-org/my-repo"
                    fontSize={"16px"}
                    padding={"10px"}
                    height={"42px"}
                    defaultValue={params.rawSource}
                    type="text"
                    {...register(FieldName.SOURCE, { required: true })}
                  />
                  <Tooltip label={"Select Folder"}>
                    <InputRightElement
                      cursor={"pointer"}
                      onClick={async () => {
                        const selected = await open({
                          directory: true,
                        })
                        if (selected) {
                          setValue(FieldName.SOURCE, selected + "", {
                            shouldDirty: true,
                          })
                        }
                      }}>
                      <Icon
                        _hover={{ color: "black" }}
                        position={"relative"}
                        top={"3px"}
                        fontSize={"18px"}
                        as={FiFile}
                        color={"grey"}
                      />
                    </InputRightElement>
                  </Tooltip>
                </InputGroup>
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
                    image={PythonSvg}
                    source={"https://github.com/microsoft/vscode-remote-try-python"}
                    currentSource={currentSource}
                    setValue={setValue}
                  />
                  <ExampleCard
                    image={NodejsPng}
                    source={"https://github.com/microsoft/vscode-remote-try-node"}
                    currentSource={currentSource}
                    setValue={setValue}
                  />
                  <ExampleCard
                    image={GoPng}
                    source={"https://github.com/Microsoft/vscode-remote-try-go"}
                    currentSource={currentSource}
                    setValue={setValue}
                  />
                  <ExampleCard
                    image={RustSvg}
                    source={"https://github.com/microsoft/vscode-remote-try-rust"}
                    currentSource={currentSource}
                    setValue={setValue}
                  />
                  <ExampleCard
                    image={JavaPng}
                    source={"https://github.com/microsoft/vscode-remote-try-java"}
                    currentSource={currentSource}
                    setValue={setValue}
                  />
                  <ExampleCard
                    image={PhpSvg}
                    source={"https://github.com/microsoft/vscode-remote-try-php"}
                    currentSource={currentSource}
                    setValue={setValue}
                  />
                  <ExampleCard
                    image={CppSvg}
                    source={"https://github.com/microsoft/vscode-remote-try-cpp"}
                    currentSource={currentSource}
                    setValue={setValue}
                  />
                  <ExampleCard
                    image={DotnetcorePng}
                    source={"https://github.com/microsoft/vscode-remote-try-dotnet"}
                    currentSource={currentSource}
                    setValue={setValue}
                  />
                </SimpleGrid>
                {exists(sourceError) ? (
                  <FormErrorMessage>{sourceError.message ?? "Error"}</FormErrorMessage>
                ) : (
                  <FormHelperText>
                    Select one of the example repositories above to get started with your favourite
                    programming language.
                  </FormHelperText>
                )}
              </FormControl>
            </TabPanel>
          </TabPanels>
        </Tabs>

        <CollapsibleSection title={"Advanced Options"} showIcon={true}>
          <VStack spacing="10" maxWidth={"1024px"}>
            <HStack spacing="8" alignItems={"top"} width={"100%"} justifyContent={"start"}>
              <FormControl isRequired isInvalid={exists(providerError)}>
                <FormLabel>Provider</FormLabel>
                <Select
                  defaultValue={params.providerID}
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
              <FormControl isRequired isInvalid={exists(defaultIDEError)}>
                <FormLabel>Default IDE</FormLabel>
                <Select
                  defaultValue={params.ide}
                  {...register(FieldName.DEFAULT_IDE, { required: true })}>
                  {idesQuery.data?.map((ide) => (
                    <option key={ide.name} value={ide.name!}>
                      {ide.displayName}
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
            </HStack>
            <FormControl isInvalid={exists(idError)}>
              <FormLabel>Workspace Name</FormLabel>
              <Input
                placeholder="my-workspace"
                type="text"
                {...register(FieldName.ID)}
                onChange={(e) => {
                  setValue(FieldName.ID, e.target.value, {
                    shouldDirty: true,
                  })

                  if (/[^a-z0-9-]+/.test(e.target.value)) {
                    setError(FieldName.ID, {
                      message: "Name can only consist of lower case letters, numbers and dashes",
                    })
                  } else {
                    clearErrors(FieldName.ID)
                  }
                }}
              />
              {exists(idError) ? (
                <FormErrorMessage>{idError.message ?? "Error"}</FormErrorMessage>
              ) : (
                <FormHelperText>
                  This is the workspace name DevPod will use. This is an optional field and usually
                  only needed if you have an already existing workspace with the same name.
                </FormHelperText>
              )}
            </FormControl>
          </VStack>
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

function useCreateWorkspaceParams() {
  const [searchParams] = useSearchParams()

  return Routes.getWorkspaceCreateParamsFromSearchParams(searchParams)
}
