import {
  Accordion,
  AccordionButton,
  AccordionIcon,
  AccordionItem,
  AccordionPanel,
  Box,
  Button,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Input,
  Select,
  VStack,
} from "@chakra-ui/react"
import { useCallback, useEffect, useMemo } from "react"
import { SubmitHandler, useForm } from "react-hook-form"
import { useNavigate } from "react-router"
import { useStreamingTerminal } from "../../components"
import { useProviders, useWorkspaceManager } from "../../contexts"
import { exists } from "../../lib"
import { Routes } from "../../routes"
import { TProviderID } from "../../types"

// https://github.com/microsoft/vscode-course-sample
const FieldName = {
  SOURCE: "source",
  DEFAULT_IDE: "defaultIDE",
  PROVIDER: "provider",
} as const

const SUPPORTED_IDES = ["vscode", "intellj"] as const
type TSupportedIDE = (typeof SUPPORTED_IDES)[number]

const DEFAULT_PROVIDER = "local"

type TFormValues = {
  [FieldName.SOURCE]: string
  [FieldName.DEFAULT_IDE]: TSupportedIDE
  [FieldName.PROVIDER]: TProviderID // TODO: needs runtime validation
}

// TODO: handle no provider configured
export function CreateWorkspace() {
  const navigate = useNavigate()
  const { create } = useWorkspaceManager()
  const [providers] = useProviders()
  const { register, handleSubmit, formState } = useForm<TFormValues>()
  const { terminal, connectStream } = useStreamingTerminal()

  const onSubmit = useCallback<SubmitHandler<TFormValues>>(
    (data) => {
      const workspaceSource = data[FieldName.SOURCE].trim()
      const providerID = data[FieldName.PROVIDER]
      const defaultIDE = data[FieldName.DEFAULT_IDE]

      // TODO: after creating a workspace, the status is NOT_FOUND until the whole devcontainer is set up...
      // can we change this in cli?
      create.run({
        rawWorkspaceSource: workspaceSource,
        config: {
          providerConfig: { providerID },
          ideConfig: { ide: defaultIDE },
          sourceConfig: {
            source: workspaceSource,
          },
        },
        onStream: connectStream,
      })
    },
    [create, connectStream]
  )

  const providerOptions = useMemo<readonly TProviderID[]>(() => {
    const maybeProviders = providers?.providers

    if (!exists(maybeProviders)) {
      return [DEFAULT_PROVIDER] // TODO: make dynamic
    }

    return Object.keys(maybeProviders)
  }, [providers])

  useEffect(() => {
    if (create.status === "success") {
      navigate(Routes.WORKSPACES)
    }
  }, [navigate, create.status])

  if (create.status === "loading") {
    return terminal
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <VStack align="start" spacing="6">
        <FormControl isRequired>
          <FormLabel>Source</FormLabel>
          <Input
            isInvalid={exists(formState.errors[FieldName.SOURCE])}
            placeholder="Source"
            type="text"
            {...register(FieldName.SOURCE, { required: true })}
          />
          <FormHelperText>
            Can be either a Git Repository, a Git Branch, a docker image or the path to a local
            folder. This cannot be changed after creating the workspace!
          </FormHelperText>
          {exists(formState.errors[FieldName.SOURCE]) && (
            <FormErrorMessage>
              {formState.errors[FieldName.SOURCE]?.message ?? "Error"}
            </FormErrorMessage>
          )}
        </FormControl>

        <FormControl isRequired>
          <FormLabel>Provider</FormLabel>
          <Select
            defaultValue={DEFAULT_PROVIDER}
            isInvalid={exists(formState.errors[FieldName.PROVIDER])}
            placeholder="Select Provider"
            {...register(FieldName.PROVIDER, { required: true })}>
            {providerOptions.map((providerID) => (
              <option key={providerID} value={providerID}>
                {providerID}
              </option>
            ))}
          </Select>
          <FormHelperText>
            Devpod will use the selected provider to spin up this workspace. This cannot be changed
            after creating the workspace!
          </FormHelperText>
          {exists(formState.errors[FieldName.PROVIDER]) && (
            <FormErrorMessage>
              {formState.errors[FieldName.PROVIDER]?.message ?? "Error"}
            </FormErrorMessage>
          )}
        </FormControl>

        <FormControl isRequired>
          <FormLabel>Default IDE</FormLabel>
          <Select
            defaultValue={"vscode"}
            isInvalid={exists(formState.errors[FieldName.DEFAULT_IDE])}
            placeholder="Select Default IDE"
            {...register(FieldName.DEFAULT_IDE, { required: true })}>
            {SUPPORTED_IDES.map((ide) => (
              <option key={ide} value={ide}>
                {ide}
              </option>
            ))}
          </Select>
          <FormHelperText>
            Devpod will open this workspace with the selected IDE by default. You can still change
            your default IDE later.
          </FormHelperText>
          {exists(formState.errors[FieldName.DEFAULT_IDE]) && (
            <FormErrorMessage>
              {formState.errors[FieldName.DEFAULT_IDE]?.message ?? "Error"}
            </FormErrorMessage>
          )}
        </FormControl>

        <Accordion allowMultiple width="full">
          <AccordionItem borderTop="none">
            <AccordionButton>
              <Box as="span" flex="1" textAlign="left">
                Advanced Options
              </Box>
              <AccordionIcon />
            </AccordionButton>
            <AccordionPanel>TODO: Implement me</AccordionPanel>
          </AccordionItem>
        </Accordion>

        <Button marginTop="10" type="submit" disabled={formState.isSubmitting}>
          Submit
        </Button>
      </VStack>
    </form>
  )
}
