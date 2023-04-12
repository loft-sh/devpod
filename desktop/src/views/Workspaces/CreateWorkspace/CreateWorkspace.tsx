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
  Select,
  Text,
  useColorModeValue,
  VStack,
  Wrap,
  WrapItem,
} from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import { useCallback, useEffect, useMemo, useState } from "react"
import { FiFolder } from "react-icons/fi"
import { useNavigate } from "react-router"
import { useSearchParams } from "react-router-dom"
import { client } from "../../../client"
import { CollapsibleSection, useStreamingTerminal } from "../../../components"
import { useProviders, useWorkspace } from "../../../contexts"
import { exists, getKeys, isEmpty, useFormErrors } from "../../../lib"
import { QueryKeys } from "../../../queryKeys"
import { Routes } from "../../../routes"
import { useBorderColor } from "../../../Theme"
import { TProviderID, TWorkspaceID } from "../../../types"
import { WORKSPACE_EXAMPLES } from "./constants"
import { ExampleCard } from ".././ExampleCard"
import { FieldName, TCreateWorkspaceArgs, TCreateWorkspaceSearchParams } from "./types"
import { useCreateWorkspaceForm } from "./useCreateWorkspaceForm"
import { useSetupProviderModal } from "./useSetupProviderModal"

export function CreateWorkspace() {
  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })

  const ides = useMemo(() => idesQuery.data, [idesQuery.data])

  const searchParams = useCreateWorkspaceParams()
  const [workspaceID, setWorkspaceID] = useState<TWorkspaceID | undefined>(undefined)
  const navigate = useNavigate()
  const workspace = useWorkspace(workspaceID)
  const [[providers]] = useProviders()
  const { terminal, connectStream } = useStreamingTerminal()

  const handleCreateWorkspace = useCallback(
    ({
      workspaceID,
      providerID,
      prebuildRepositories,
      defaultIDE,
      workspaceSource,
    }: TCreateWorkspaceArgs) => {
      workspace.create(
        {
          id: workspaceID,
          prebuildRepositories,
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
    [connectStream, workspace]
  )

  const {
    setValue,
    register,
    onSubmit,
    validateWorkspaceID,
    formState,
    isSubmitting,
    currentSource,
  } = useCreateWorkspaceForm(searchParams, providers, ides, handleCreateWorkspace)
  const { sourceError, providerError, defaultIDEError, idError, prebuildRepositoryError } =
    useFormErrors(Object.values(FieldName), formState)

  const providerOptions = useMemo<readonly TProviderID[]>(() => {
    if (!exists(providers)) {
      return [] // TODO: make dynamic
    }

    return Object.keys(providers)
  }, [providers])

  const isLoading = useMemo(() => workspace.current?.name === "create", [workspace])

  const handleSelectFolderClicked = useCallback(async () => {
    const selected = await client.selectFromDir()
    if (typeof selected === "string") {
      setValue(FieldName.SOURCE, selected, {
        shouldDirty: true,
      })
    }
  }, [setValue])

  const handleExampleCardClicked = useCallback(
    (newSource: string) => {
      setValue(FieldName.SOURCE, newSource, {
        shouldDirty: true,
      })
    },
    [setValue]
  )

  const { modal, show: showSetupProviderModal, wasDismissed } = useSetupProviderModal()
  useEffect(() => {
    if (wasDismissed) {
      return
    }

    if (providers !== undefined) {
      // no provider available
      if (isEmpty(getKeys(providers))) {
        showSetupProviderModal({
          message: "Looks like you don't have providers installed yet.",
          isStrict: true,
        })

        return
      }

      // selected provider not installed
      if (searchParams.providerID && providers[searchParams.providerID] === undefined) {
        showSetupProviderModal({
          message: `You tried to create a workspace with the "${searchParams.providerID}" provider. It looks like this provider isn't available on your machine. 
          Please set it up first. Alternatively you can create a workspace with a different provider.`,
          isStrict: false,
        })

        return
      }
    }
  }, [providers, searchParams.providerID, showSetupProviderModal, wasDismissed])

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
    <>
      <form onSubmit={onSubmit}>
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
              justifyContent={"center"}
              borderBottomWidth="thin"
              borderBottomColor={borderColor}>
              <VStack>
                <Text marginBottom="2" fontWeight="bold">
                  Enter any git repository or local path to a folder you would like to create a
                  workspace from
                </Text>
                <HStack spacing={0} justifyContent={"center"}>
                  <Input
                    backgroundColor={inputBackgroundColor}
                    borderTopRightRadius={0}
                    borderBottomRightRadius={0}
                    placeholder="github.com/my-org/my-repo"
                    fontSize={"16px"}
                    padding={"10px"}
                    height={"42px"}
                    width={"400px"}
                    type="text"
                    {...register(FieldName.SOURCE, { required: true })}
                  />
                  <Button
                    leftIcon={<Icon as={FiFolder} />}
                    borderTopLeftRadius={0}
                    borderBottomLeftRadius={0}
                    borderTop={"1px solid white"}
                    borderRight={"1px solid white"}
                    borderBottom={"1px solid white"}
                    borderColor={"gray.200"}
                    height={"42px"}
                    flex={"0 0 140px"}
                    onClick={handleSelectFolderClicked}>
                    Select Folder
                  </Button>
                </HStack>
                {exists(sourceError) && (
                  <FormErrorMessage>{sourceError.message ?? "Error"}</FormErrorMessage>
                )}
              </VStack>
            </FormControl>

            <Box width="full" height="full" padding={4} marginBottom="8">
              <CollapsibleSection title="Or use one of our quickstart examples" showIcon isOpen>
                <FormControl isRequired isInvalid={exists(sourceError)}>
                  <Wrap spacing={3} marginTop="2.5" justify="center">
                    {WORKSPACE_EXAMPLES.map((example) => (
                      <WrapItem key={example.source} padding={"1"}>
                        <ExampleCard
                          image={example.image}
                          source={example.source}
                          isSelected={currentSource === example.source}
                          onClick={handleExampleCardClicked}
                        />
                      </WrapItem>
                    ))}
                  </Wrap>
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
                      DevPod will open this workspace with the selected IDE by default. You can
                      still change your default IDE later.
                    </FormHelperText>
                  )}
                </FormControl>
              </HStack>
              <HStack spacing="8" alignItems={"top"} width={"100%"} justifyContent={"start"}>
                <FormControl isInvalid={exists(idError)}>
                  <FormLabel>Workspace Name</FormLabel>
                  <Input
                    placeholder="my-workspace"
                    type="text"
                    {...register(FieldName.ID)}
                    onChange={validateWorkspaceID}
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
                    placeholder="ghcr.io/my-org/my-repo"
                    type="text"
                    {...register(FieldName.PREBUILD_REPOSITORY)}
                    onChange={(e) => {
                      setValue(FieldName.PREBUILD_REPOSITORY, e.target.value, {
                        shouldDirty: true,
                      })
                    }}
                  />
                  {exists(prebuildRepositoryError) ? (
                    <FormErrorMessage>
                      {prebuildRepositoryError.message ?? "Error"}
                    </FormErrorMessage>
                  ) : (
                    <FormHelperText>
                      DevPod will use this repository to find prebuilds for the given workspace.
                    </FormHelperText>
                  )}
                </FormControl>
              </HStack>
            </VStack>
          </CollapsibleSection>

          <Button
            variant="primary"
            marginTop="10"
            type="submit"
            disabled={formState.isSubmitting}
            isLoading={isSubmitting}>
            Create Workspace
          </Button>
        </VStack>
      </form>

      {modal}
    </>
  )
}

function useCreateWorkspaceParams(): TCreateWorkspaceSearchParams {
  const [searchParams] = useSearchParams()

  return useMemo(
    () => Routes.getWorkspaceCreateParamsFromSearchParams(searchParams),
    [searchParams]
  )
}
