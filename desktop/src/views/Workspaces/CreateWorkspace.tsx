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
import { exists, useFormErrors } from "../../lib"
import { Routes } from "../../routes"
import { TProviderID } from "../../types"

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
  const [[providers]] = useProviders()
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

  const { sourceError, defaultIDEError, providerError } = useFormErrors(
    Object.values(FieldName),
    formState
  )

  const providerOptions = useMemo<readonly TProviderID[]>(() => {
    if (!exists(providers)) {
      return [DEFAULT_PROVIDER] // TODO: make dynamic
    }

    return Object.keys(providers)
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
        <FormControl isRequired isInvalid={exists(sourceError)}>
          <FormLabel>Source</FormLabel>
          <Input
            placeholder="Source"
            type="text"
            {...register(FieldName.SOURCE, { required: true })}
          />
          {exists(sourceError) ? (
            <FormErrorMessage>{sourceError.message ?? "Error"}</FormErrorMessage>
          ) : (
            <FormHelperText>
              Can be either a Git Repository, a Git Branch, a docker image or the path to a local
              folder. This cannot be changed after creating the workspace!
            </FormHelperText>
          )}
        </FormControl>

        <FormControl isRequired isInvalid={exists(providerError)}>
          <FormLabel>Provider</FormLabel>
          <Select
            defaultValue={DEFAULT_PROVIDER}
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
            <FormHelperText>
              Devpod will use the selected provider to spin up this workspace. This cannot be
              changed after creating the workspace!
            </FormHelperText>
          )}
        </FormControl>

        <FormControl isRequired isInvalid={exists(defaultIDEError)}>
          <FormLabel>Default IDE</FormLabel>
          <Select
            defaultValue={"vscode"}
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
              Devpod will open this workspace with the selected IDE by default. You can still change
              your default IDE later.
            </FormHelperText>
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
