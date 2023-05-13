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
  SimpleGrid,
  Text,
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
import { RECOMMENDED_PROVIDER_SOURCES, SIDEBAR_WIDTH } from "../../../constants"
import { useProviders, useWorkspace } from "../../../contexts"
import { exists, getKeys, isEmpty, useFormErrors } from "../../../lib"
import { QueryKeys } from "../../../queryKeys"
import { Routes } from "../../../routes"
import { useBorderColor } from "../../../Theme"
import { TIDE, TProviderID } from "../../../types"
import { useSetupProviderModal } from "../../Providers"
import { WORKSPACE_EXAMPLES } from "./constants"
import {
  FieldName,
  TCreateWorkspaceArgs,
  TCreateWorkspaceSearchParams,
  TFormValues,
  TSelectProviderOptions,
} from "./types"
import { useCreateWorkspaceForm } from "./useCreateWorkspaceForm"

const Form = styled.form`
  width: 100%;
  display: flex;
  justify-content: center;
`

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

  const { setValue, register, onSubmit, control, formState, isSubmitting, currentSource } =
    useCreateWorkspaceForm(searchParams, providers, ides, handleCreateWorkspace)
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
        shouldValidate: true,
      })
    }
  }, [setValue])

  const handleExampleCardClicked = useCallback(
    (newSource: string) => {
      setValue(FieldName.SOURCE, newSource, {
        shouldDirty: true,
        shouldValidate: true,
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
        // showSetupProviderModal({ isStrict: false })

        return
      }
    }
  }, [providers, searchParams.providerID, showSetupProviderModal, wasDismissed])

  const backgroundColor = useColorModeValue("gray.50", "gray.800")
  const borderColor = useBorderColor()
  const inputBackgroundColor = useColorModeValue("white", "black")

  return (
    <>
      <Form onSubmit={onSubmit}>
        <VStack align="start" spacing="6" alignItems="center" width="full" maxWidth="container.lg">
          <HStack
            width="full"
            height="full"
            borderRadius="lg"
            borderWidth="thin"
            borderColor={borderColor}>
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
                      name={example.name}
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
                  {...register(FieldName.ID, {
                    validate: {
                      name: (value) => {
                        if (/[^a-z0-9-]+/.test(value)) {
                          return "Name can only consist of lower case letters, numbers and dashes"
                        } else {
                          return true
                        }
                      },
                    },
                    maxLength: { value: 48, message: "Name cannot be longer than 48 characters" },
                  })}
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
          </VStack>

          <HStack
            position="sticky"
            bottom="-1.1rem"
            right="0"
            width={{ base: `calc(100vw - ${SIDEBAR_WIDTH})`, xl: "full" }}
            height="20"
            alignItems="center"
            borderTopWidth="thin"
            borderTopColor={borderColor}
            backgroundColor="white"
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
          <Box key={p.name}>
            <ExampleCard
              isSelected={field.value === p.name}
              name={p.name}
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
        ))}
      </HStack>
      <HStack>
        {options.recommended.map((p) => (
          <Box key={p.name} filter="grayscale(100%)" _hover={{ filter: "grayscale(0%)" }}>
            <ExampleCard
              name={p.name}
              size="sm"
              onClick={() => onRecommendedProviderClicked(p.name)}
              image={p.image}
            />
          </Box>
        ))}
      </HStack>
    </Grid>
  )
}

import { NoneSvg } from "../../../images"
import styled from "@emotion/styled"
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
          <Box key={ide.name}>
            <ExampleCard
              name={ide.displayName}
              size="sm"
              image={ide.icon ?? NoneSvg}
              isSelected={isSelected}
              onClick={() => onClick(ide.name!)}
            />
          </Box>
        )
      })}
    </Grid>
  )
}
