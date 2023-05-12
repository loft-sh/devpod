import {
  Box,
  Button,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Grid,
  HStack,
  Icon,
  Input,
  Link,
  Select,
  SimpleGrid,
  Text,
  Tooltip,
  useColorModeValue,
  useToken,
  VStack,
} from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import { useCallback, useEffect, useMemo } from "react"
import { Controller, ControllerRenderProps } from "react-hook-form"
import { FiFolder } from "react-icons/fi"
import { useNavigate } from "react-router"
import { useSearchParams } from "react-router-dom"
import { client } from "../../../client"
import { ExampleCard } from "../../../components"
import { RECOMMENDED_PROVIDER_SOURCES, SIDEBAR_WIDTH, STATUS_BAR_HEIGHT } from "../../../constants"
import { useProviders, useWorkspace } from "../../../contexts"
import { exists, getIDEDisplayName, getKeys, isEmpty, useFormErrors } from "../../../lib"
import { QueryKeys } from "../../../queryKeys"
import { Routes } from "../../../routes"
import { useBorderColor } from "../../../Theme"
import { TIDE, TProviderID } from "../../../types"
import { WORKSPACE_EXAMPLES } from "./constants"
import {
  FieldName,
  TCreateWorkspaceArgs,
  TCreateWorkspaceSearchParams,
  TFormValues,
  TSelectProviderOptions,
} from "./types"
import { useCreateWorkspaceForm } from "./useCreateWorkspaceForm"
import { useSetupProviderModal } from "./useSetupProviderModal"

export function CreateWorkspace() {
  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })

  const ides = useMemo(() => idesQuery.data, [idesQuery.data])

  const searchParams = useCreateWorkspaceParams()
  const navigate = useNavigate()
  const workspace = useWorkspace(undefined)
  const [[providers]] = useProviders()

  const handleCreateWorkspace = useCallback(
    ({
      workspaceID,
      providerID,
      prebuildRepositories,
      defaultIDE,
      workspaceSource,
    }: TCreateWorkspaceArgs) => {
      const actionID = workspace.create({
        id: workspaceID,
        prebuildRepositories,
        providerConfig: { providerID },
        ideConfig: { name: defaultIDE },
        sourceConfig: {
          source: workspaceSource,
        },
      })

      // set workspace id to show terminal
      if (!isEmpty(actionID)) {
        navigate(Routes.toAction(actionID, Routes.WORKSPACES))
      }
    },
    [navigate, workspace]
  )

  const {
    setValue,
    register,
    onSubmit,
    validateWorkspaceID,
    control,
    formState,
    isSubmitting,
    currentSource,
  } = useCreateWorkspaceForm(searchParams, providers, ides, handleCreateWorkspace)
  const { sourceError, providerError, defaultIDEError, idError, prebuildRepositoryError } =
    useFormErrors(Object.values(FieldName), formState)

  const providerOptions = useMemo<TSelectProviderOptions>(() => {
    if (!exists(providers)) {
      return { installed: [], recommended: RECOMMENDED_PROVIDER_SOURCES }
    }

    return {
      installed: Object.entries(providers).map(([key, value]) => ({ name: key, ...value })),
      recommended: RECOMMENDED_PROVIDER_SOURCES,
    }
  }, [providers])

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
        showSetupProviderModal({ isStrict: true })

        return
      }

      // selected provider not installed
      if (searchParams.providerID && providers[searchParams.providerID] === undefined) {
        showSetupProviderModal({ isStrict: false })

        return
      }
    }
  }, [providers, searchParams.providerID, showSetupProviderModal, wasDismissed])

  const backgroundColor = useColorModeValue("gray.50", "gray.800")
  const borderColor = useBorderColor()
  const inputBackgroundColor = useColorModeValue("white", "black")

  return (
    <>
      <form onSubmit={onSubmit}>
        <VStack align="start" spacing="6" marginBottom="8" alignItems="center" width="full">
          <HStack
            width="full"
            height="full"
            borderRadius="lg"
            borderWidth="thin"
            borderColor={borderColor}
            maxWidth="container.lg">
            <FormControl
              backgroundColor={backgroundColor}
              paddingX="20"
              paddingY="32"
              height="full"
              isRequired
              isInvalid={exists(sourceError)}
              justifyContent="center"
              display="flex"
              borderRightWidth="thin"
              borderRightColor={borderColor}>
              <VStack width="full">
                <Text marginBottom="2" fontWeight="bold">
                  Enter Workspace Source
                </Text>
                <HStack spacing={0} justifyContent={"center"} width="full">
                  <Input
                    spellCheck={false}
                    backgroundColor={inputBackgroundColor}
                    borderTopRightRadius={0}
                    borderBottomRightRadius={0}
                    placeholder="github.com/my-org/my-repo"
                    fontSize={"16px"}
                    padding={"10px"}
                    height={"42px"}
                    width={"60%"}
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
                <FormHelperText textAlign={"center"}>
                  Any git repository or local path to a folder you would like to create a workspace
                  from can be a source as long as it adheres to the{" "}
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
                color="gray.500"
                marginBottom="4"
                fontWeight="medium"
                textAlign="center">
                Or select one of our quickstart examples
              </Text>
              <FormControl
                paddingTop="6"
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
                      image={example.image}
                      source={example.source}
                      isSelected={currentSource === example.source}
                      onClick={() => handleExampleCardClicked(example.source)}
                    />
                  ))}
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
                  render={({ field }) => (
                    <ProviderInput
                      field={field}
                      options={providerOptions}
                      onRecommendedProviderClicked={(name) =>
                        showSetupProviderModal({
                          suggestedProvider: name,
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
                    <IDEInput
                      field={field}
                      ides={idesQuery.data}
                      onClick={(name) => field.onChange(name)}
                    />
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
                  spellCheck={false}
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
                  <FormErrorMessage>{prebuildRepositoryError.message ?? "Error"}</FormErrorMessage>
                ) : (
                  <FormHelperText>
                    DevPod will use this repository to find prebuilds for the given workspace.
                  </FormHelperText>
                )}
              </FormControl>
            </HStack>
          </VStack>

          <HStack
            position="absolute"
            bottom={STATUS_BAR_HEIGHT}
            right="0"
            width={`calc(100vw - ${SIDEBAR_WIDTH})`}
            height="20"
            backgroundColor="white"
            alignItems="center"
            borderTopWidth="thin"
            borderTopColor={borderColor}
            paddingX="8"
            zIndex="overlay">
            <Button
              variant="primary"
              type="submit"
              disabled={formState.isSubmitting}
              isLoading={isSubmitting}>
              Create Workspace
            </Button>
          </HStack>
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

type TProviderInputProps = Readonly<{
  options: TSelectProviderOptions
  field: ControllerRenderProps<TFormValues, (typeof FieldName)["PROVIDER"]>
  onRecommendedProviderClicked: (provider: TProviderID) => void
}>
function ProviderInput({ options, field, onRecommendedProviderClicked }: TProviderInputProps) {
  return (
    <Grid
      templateRows={{
        lg: "repeat(2, 1fr)",
        xl: options.installed.length <= 2 ? "1fr" : "repeat(2, 1fr)",
      }}
      gridAutoFlow={"column"}>
      <HStack>
        {options.installed.map((p) => (
          <Tooltip key={p.name} label={p.name}>
            <Box>
              <ExampleCard
                isSelected={field.value === p.name}
                size="sm"
                onClick={() => field.onChange(p.name)}
                image={p.config?.icon ?? undefined}
              />
              <Text
                maxWidth="10"
                overflow="hidden"
                textOverflow="ellipsis"
                whiteSpace="nowrap"
                textAlign="center"
                fontSize="sm"
                color="gray.500">
                {p.name}
              </Text>
            </Box>
          </Tooltip>
        ))}
      </HStack>
      <HStack>
        {options.recommended.map((p) => (
          <Tooltip key={p.name} label={p.name}>
            <Box filter="grayscale(100%)" _hover={{ filter: "grayscale(0%)" }}>
              <ExampleCard
                size="sm"
                onClick={() => onRecommendedProviderClicked(p.name)}
                image={p.image}
              />
              <Text
                maxWidth="10"
                overflow="hidden"
                textOverflow="ellipsis"
                whiteSpace="nowrap"
                textAlign="center"
                fontSize="sm"
                color="gray.500">
                {p.name}
              </Text>
            </Box>
          </Tooltip>
        ))}
      </HStack>
    </Grid>
  )
}

import { KubernetesSvg } from "../../../images"
type TIDEInputProps = Readonly<{
  ides: readonly TIDE[] | undefined
  field: ControllerRenderProps<TFormValues, (typeof FieldName)["DEFAULT_IDE"]>
  onClick: (name: TIDE["name"]) => void
}>
function IDEInput({ ides, field, onClick }: TIDEInputProps) {
  const gridChildWidth = useToken("sizes", "12")

  return (
    <Grid
      gridTemplateColumns={{
        lg: `repeat(8, ${gridChildWidth})`,
        xl: `repeat(11, ${gridChildWidth})`,
        "2xl": `repeat(auto-fit, ${gridChildWidth})`,
      }}>
      {ides?.map((ide) => {
        const isSelected = field.value === ide.name

        return (
          <Box
            key={ide.name}
            >
            <ExampleCard
              size="sm"
              image={ide.icon ?? KubernetesSvg}
              isSelected={isSelected}
              onClick={() => onClick(ide.name!)}
            />
          </Box>
        )
      })}
    </Grid>
  )
}
