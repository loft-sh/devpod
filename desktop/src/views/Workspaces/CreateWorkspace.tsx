import {
  Box,
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
  Text,
  Tooltip,
  useColorModeValue,
  VStack,
} from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import { useCallback, useEffect, useMemo, useState } from "react"
import { SubmitHandler, useForm } from "react-hook-form"
import { FiFile } from "react-icons/fi"
import { useNavigate } from "react-router"
import { useSearchParams } from "react-router-dom"
import { DEFAULT_ECDH_CURVE } from "tls"
import { client } from "../../client"
import { CollapsibleSection, useStreamingTerminal } from "../../components"
import { useProviders, useWorkspace, useWorkspaces } from "../../contexts"
import {
  CppSvg,
  DotnetcorePng,
  GoPng,
  JavaPng,
  NodejsPng,
  PhpSvg,
  PythonSvg,
  RustSvg,
} from "../../images"
import { exists, useFormErrors } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import { Routes } from "../../routes"
import { useBorderColor } from "../../Theme"
import { TProviderID, TWorkspaceID } from "../../types"
import { ExampleCard } from "./ExampleCard"
import { FieldName, TFormValues } from "./types"

const DEFAULT_PROVIDER = "docker"

// TODO: handle no provider configured
export function CreateWorkspace() {
  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })

  const [isSubmitLoading, setIsSubmitLoading] = useState(false)
  const params = useCreateWorkspaceParams()
  const [workspaceID, setWorkspaceID] = useState<TWorkspaceID | undefined>(undefined)
  const navigate = useNavigate()
  const workspaces = useWorkspaces()
  const workspace = useWorkspace(workspaceID)
  const [[providers]] = useProviders()
  const { register, handleSubmit, formState, watch, setError, setValue, clearErrors, reset } =
    useForm<TFormValues>({
      defaultValues: {
        [FieldName.PROVIDER]: DEFAULT_PROVIDER,
        [FieldName.DEFAULT_IDE]: "vscode",
      },
    })
  const currentSource = watch(FieldName.SOURCE)
  const { terminal, connectStream } = useStreamingTerminal()

  useEffect(() => {
    reset({
      ...(params.rawSource !== undefined ? { [FieldName.SOURCE]: params.rawSource } : {}),
      ...(params.ide !== undefined ? { [FieldName.DEFAULT_IDE]: params.ide } : {}),
      ...(params.providerID !== undefined ? { [FieldName.PROVIDER]: params.providerID } : {}),
    })
  }, [params, reset])

  const onSubmit = useCallback<SubmitHandler<TFormValues>>(
    async (data) => {
      const workspaceSource = data[FieldName.SOURCE].trim()
      setIsSubmitLoading(true)
      let workspaceID = data[FieldName.ID]
      if (!workspaceID) {
        const newIDResult = await client.workspaces.newID(workspaceSource)
        if (newIDResult.err) {
          setIsSubmitLoading(false)
          setError(FieldName.SOURCE, { message: newIDResult.val.message })

          return
        }

        workspaceID = newIDResult.val
      }
      setIsSubmitLoading(false)

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

  const isLoading = useMemo(() => workspace.current?.name === "create", [workspace])

  const handleSelectFolderClicked = useCallback(async () => {
    const selected = await client.selectFromDir()
    if (selected) {
      setValue(FieldName.SOURCE, selected + "", {
        shouldDirty: true,
      })
    }
  }, [setValue])

  useEffect(() => {
    if (
      workspace.current?.name === "create" &&
      workspace.current.status === "success" &&
      workspace.data?.id !== undefined
    ) {
      navigate(Routes.WORKSPACES)
    }
  }, [navigate, workspace])

  const backgroundColor = useColorModeValue("blackAlpha.100", "whiteAlpha.100")
  const borderColor = useBorderColor()
  const inputBackgroundColor = useColorModeValue("white", "black")

  if (isLoading) {
    return terminal
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <VStack align="start" spacing="6" marginBottom="8">
        <VStack
          width="full"
          backgroundColor={backgroundColor}
          borderRadius="lg"
          borderWidth="thin"
          borderColor={borderColor}>
          <FormControl
            padding="20"
            isRequired
            isInvalid={exists(sourceError)}
            borderBottomWidth="thin"
            borderBottomColor={borderColor}>
            <Text marginBottom="2" fontWeight="bold">
              Enter any git repository or local path to a folder you would like to create a
              workspace from
            </Text>
            <InputGroup backgroundColor={inputBackgroundColor} borderRadius="md">
              <Input
                placeholder="github.com/my-org/my-repo"
                fontSize={"16px"}
                padding={"10px"}
                height={"42px"}
                type="text"
                {...register(FieldName.SOURCE, { required: true })}
              />
              <Tooltip label={"Select Folder"}>
                <InputRightElement cursor={"pointer"} onClick={handleSelectFolderClicked}>
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
              <FormHelperText></FormHelperText>
            )}
          </FormControl>

          <Box width="full" height="full" padding={4} marginBottom="8">
            <CollapsibleSection title="Or use one of our quickstart examples" showIcon isOpen>
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
                  <FormHelperText></FormHelperText>
                )}
              </FormControl>
            </CollapsibleSection>
          </Box>
        </VStack>

        <CollapsibleSection title={"Advanced Options"} showIcon>
          <VStack spacing="10" maxWidth={"1024px"}>
            <HStack spacing="8" alignItems={"top"} width={"100%"} justifyContent={"start"}>
              <FormControl isRequired isInvalid={exists(providerError)}>
                <FormLabel>Provider</FormLabel>
                <Select {...register(FieldName.PROVIDER)}>
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
                <Select {...register(FieldName.DEFAULT_IDE)}>
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
          variant="primary"
          marginTop="10"
          type="submit"
          disabled={formState.isSubmitting}
          isLoading={formState.isSubmitting || isSubmitLoading}>
          Create Workspace
        </Button>
      </VStack>
    </form>
  )
}

function useCreateWorkspaceParams() {
  const [searchParams] = useSearchParams()

  return useMemo(
    () => Routes.getWorkspaceCreateParamsFromSearchParams(searchParams),
    [searchParams]
  )
}
