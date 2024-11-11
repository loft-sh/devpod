import { BottomActionBar, BottomActionBarError, Form } from "@/components"
import { ProWorkspaceInstance } from "@/contexts"
import { Code, Laptop, Parameters } from "@/icons"
import {
  Annotations,
  Failed,
  Labels,
  Source,
  exists,
  getParametersWithValues,
  useFormErrors,
} from "@/lib"
import { useIDEs } from "@/useIDEs"
import {
  Box,
  Button,
  ButtonGroup,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Grid,
  Input,
  Spinner,
  VStack,
} from "@chakra-ui/react"
import { ManagementV1DevPodWorkspaceTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceTemplate"
import { ReactNode, useEffect, useMemo, useRef } from "react"
import { Controller, DefaultValues, FormProvider, useForm } from "react-hook-form"
import { DevContainerInput } from "./DevContainerInput"
import { IDEInput } from "./IDEInput"
import { OptionsInput } from "./OptionsInput"
import { SourceInput } from "./SourceInput"
import { FieldName, TFormValues } from "./types"
import { useTemplates } from "@/contexts"

type TCreateWorkspaceFormProps = Readonly<{
  instance?: ProWorkspaceInstance
  template?: ManagementV1DevPodWorkspaceTemplate
  onSubmit: (data: TFormValues) => void
  onReset: VoidFunction
  error: Failed | null
}>
export function CreateWorkspaceForm({
  instance,
  template,
  onSubmit,
  onReset,
  error,
}: TCreateWorkspaceFormProps) {
  const defaultValues = useMemo(() => getDefaultValues(instance, template), [instance, template])
  const containerRef = useRef<HTMLDivElement>(null)
  const { ides, defaultIDE } = useIDEs()
  const { data: templates, isLoading: isTemplatesLoading } = useTemplates()
  const form = useForm<TFormValues>({ mode: "onChange", defaultValues })
  const { sourceError, defaultIDEError, nameError, devcontainerJSONError, optionsError } =
    useFormErrors(Object.values(FieldName), form.formState)

  useEffect(() => {
    if (!form.getFieldState(FieldName.DEFAULT_IDE).isDirty && defaultIDE && defaultIDE.name) {
      form.setValue(FieldName.DEFAULT_IDE, defaultIDE.name, {
        shouldDirty: false,
        shouldTouch: true,
      })
    }
  }, [defaultIDE, form])

  return (
    <Form onSubmit={form.handleSubmit(onSubmit)}>
      <FormProvider {...form}>
        <VStack w="full" gap="8" ref={containerRef}>
          <FormControl isDisabled={!!instance} isRequired isInvalid={exists(sourceError)}>
            <CreateWorkspaceRow
              label={
                <FormLabel>
                  <Code boxSize={5} mr="1" />
                  Source Code
                </FormLabel>
              }>
              <SourceInput isDisabled={!!instance} />

              {exists(sourceError) && (
                <FormErrorMessage>{sourceError.message ?? "Error"}</FormErrorMessage>
              )}
            </CreateWorkspaceRow>
          </FormControl>

          <FormControl isRequired isInvalid={exists(optionsError)}>
            <CreateWorkspaceRow
              label={
                <FormLabel>
                  <Parameters boxSize={5} mr="1" />
                  Parameters
                </FormLabel>
              }>
              {isTemplatesLoading ? (
                <Spinner />
              ) : (
                <OptionsInput
                  workspaceTemplates={templates!.workspace}
                  defaultWorkspaceTemplate={templates!.default}
                />
              )}

              {exists(optionsError) && (
                <FormErrorMessage>{optionsError.message ?? "Error"}</FormErrorMessage>
              )}
            </CreateWorkspaceRow>
          </FormControl>

          <FormControl isInvalid={exists(defaultIDEError)}>
            <CreateWorkspaceRow
              label={
                <VStack gap="1" align="start">
                  <FormLabel>
                    <Laptop boxSize={5} mr="1" />
                    Default IDE
                  </FormLabel>
                  <FormHelperText mt="0">
                    The default IDE to use when starting the workspace. This can be changed later.
                  </FormHelperText>
                </VStack>
              }>
              <Controller
                name={FieldName.DEFAULT_IDE}
                control={form.control}
                render={({ field }) => (
                  <IDEInput field={field} ides={ides} onClick={(name) => field.onChange(name)} />
                )}
              />
              {exists(defaultIDEError) && (
                <FormErrorMessage>{defaultIDEError.message ?? "Error"}</FormErrorMessage>
              )}
            </CreateWorkspaceRow>
          </FormControl>

          <FormControl isDisabled={!!instance} isInvalid={exists(devcontainerJSONError)}>
            <CreateWorkspaceRow
              label={
                <VStack gap="1" align="start">
                  <FormLabel>
                    <Laptop boxSize={5} mr="1" />
                    Devcontainer.json
                  </FormLabel>
                  <FormHelperText mt="0">
                    Set an external source or a relative path in the source code. Otherwise, we’ll
                    look in the code repository.
                  </FormHelperText>
                </VStack>
              }>
              <DevContainerInput environmentTemplates={templates?.environment ?? []} />

              {exists(devcontainerJSONError) && (
                <FormErrorMessage>{devcontainerJSONError.message ?? "Error"}</FormErrorMessage>
              )}
            </CreateWorkspaceRow>
          </FormControl>

          <FormControl isInvalid={exists(nameError)}>
            <CreateWorkspaceRow
              label={
                <FormLabel>
                  <Laptop boxSize={5} mr="1" />
                  Workspace Name
                </FormLabel>
              }>
              <Input {...form.register(FieldName.NAME, { required: false })} bg="white" />

              {exists(nameError) && (
                <FormErrorMessage>{nameError.message ?? "Error"}</FormErrorMessage>
              )}
            </CreateWorkspaceRow>
          </FormControl>

          <BottomActionBar hasSidebar={false} stickToBottom>
            <BottomActionBarError error={error} containerRef={containerRef} />
            <ButtonGroup marginLeft="auto">
              <Button
                isDisabled={Object.keys(form.formState.dirtyFields).length === 0}
                onClick={() => {
                  form.reset(defaultValues)
                  onReset()
                }}>
                {instance ? "Reset Changes" : "Cancel"}{" "}
              </Button>
              <Button
                type="submit"
                isLoading={form.formState.isSubmitting}
                isDisabled={
                  Object.keys(form.formState.errors).length > 0 ||
                  Object.keys(form.formState.dirtyFields).length === 0
                }>
                {instance ? "Save & Rebuild" : "Create Workspace"}
              </Button>
            </ButtonGroup>
          </BottomActionBar>
        </VStack>
      </FormProvider>
    </Form>
  )
}

type TCreateWorkspaceRowProps = Readonly<{
  label: ReactNode
  children: ReactNode
}>
function CreateWorkspaceRow({ label, children }: TCreateWorkspaceRowProps) {
  return (
    <Grid templateColumns="1fr 3fr" w="full">
      <Box w="full" h="full" pr="10">
        {label}
      </Box>
      <Box w="full" h="full">
        {children}
      </Box>
    </Grid>
  )
}

function getDefaultValues(
  instance: ProWorkspaceInstance | undefined,
  template: ManagementV1DevPodWorkspaceTemplate | undefined
): DefaultValues<TFormValues> | undefined {
  if (instance === undefined) {
    return undefined
  }
  const defaultValues: DefaultValues<TFormValues> = {
    defaultIDE: instance.status?.ide?.name ?? "none",
  }

  // source
  const rawSource = instance.metadata?.annotations?.[Annotations.WorkspaceSource]
  if (rawSource) {
    const source = Source.fromRaw(rawSource)
    defaultValues.sourceType = source.type
    defaultValues.source = source.value
  }

  // infrastructure template
  if (template && instance.spec?.parameters) {
    if (!defaultValues.options) {
      defaultValues.options = {}
    }
    defaultValues.options.workspaceTemplate = instance.spec.templateRef?.name
    defaultValues.options.workspaceTemplateVersion = instance.spec.templateRef?.version

    const parameters = getParametersWithValues(instance, template)
    if (parameters && parameters.length > 0) {
      for (const parameter of parameters) {
        if (!parameter.variable) {
          continue
        }
        // dirty, dirty hack, maybe come back and fix types
        defaultValues.options[parameter.variable] = parameter.value as any
      }
    }
  }

  // environment template
  const environmentRefName = instance.spec?.environmentRef?.name
  if (environmentRefName) {
    defaultValues.devcontainerType = "external"
    defaultValues.devcontainerJSON = environmentRefName
  }

  // name
  const name = instance.spec?.displayName ?? instance.metadata?.labels?.[Labels.WorkspaceID]
  if (name) {
    defaultValues.name = name
  }

  return defaultValues
}
