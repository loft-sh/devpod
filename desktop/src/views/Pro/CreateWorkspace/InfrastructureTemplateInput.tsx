import { useBorderColor } from "@/Theme"
import { exists, getDisplayName, getParameters, sortByVersionDesc } from "@/lib"
import { TProviderOption } from "@/types"
import { TOptionWithID } from "@/views/Providers"
import {
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Input,
  Select,
  SimpleGrid,
  Switch,
  Textarea,
  VStack,
  useColorModeValue,
} from "@chakra-ui/react"
import { ManagementV1DevPodWorkspaceTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceTemplate"
import { StorageV1AppParameter } from "@loft-enterprise/client/gen/models/storageV1AppParameter"
import { ReactNode, useMemo } from "react"
import { ChangeHandler, Controller, useFormContext } from "react-hook-form"
import { FieldName, TFormValues } from "./types"

type TOptionsInputProps = Readonly<{
  resetPreset?: VoidFunction
  infraTemplates: readonly ManagementV1DevPodWorkspaceTemplate[]
  defaultInfraTemplate: ManagementV1DevPodWorkspaceTemplate | undefined
}>
export function InfrastructureTemplateInput({
  infraTemplates: templates,
  defaultInfraTemplate,
  resetPreset,
}: TOptionsInputProps) {
  const { getValues, watch, resetField, formState } = useFormContext<TFormValues>()
  const borderColor = useBorderColor()

  const defaultTemplate = defaultInfraTemplate ?? templates[0]
  const selectedTemplateName = watch(
    `${FieldName.OPTIONS}.workspaceTemplate`,
    defaultTemplate?.metadata?.name
  )
  const selectedTemplateVersion = watch(`${FieldName.OPTIONS}.workspaceTemplateVersion`)
  const currentTemplate = useMemo(
    () => templates.find((template) => template.metadata?.name === selectedTemplateName),
    [selectedTemplateName, templates]
  )
  const currentParameters = useMemo(() => {
    let v = selectedTemplateVersion
    if (selectedTemplateVersion === "latest") {
      v = ""
    }

    return getParameters(currentTemplate, v)
  }, [currentTemplate, selectedTemplateVersion])

  const currentTemplateVersions = useMemo(() => {
    return currentTemplate?.spec?.versions?.slice().sort(sortByVersionDesc)
  }, [currentTemplate?.spec?.versions])

  const resetTemplate = () => {
    resetPreset?.()

    // reset all other options, including version
    const options = getValues("options")
    for (const [k] of Object.entries(options)) {
      if (k === "workspaceTemplate") {
        continue
      }
      if (k === "workspaceTemplateVersion") {
        resetField(`${FieldName.OPTIONS}.workspaceTemplateVersion`, {
          defaultValue: "latest",
        })
        continue
      }
      resetField(`${FieldName.OPTIONS}.${k}`, {})
    }
  }

  const resetTemplateVersion = () => {
    resetPreset?.()

    const resetOptions: Parameters<typeof resetField>[1] = {
      defaultValue: undefined,
    }
    // reset all parameters options
    const options = getValues("options")
    for (const [k] of Object.entries(options)) {
      if (k === "workspaceTemplate" || k === "workspaceTemplateVersion") {
        continue
      }
      resetField(`${FieldName.OPTIONS}.${k}`, resetOptions)
    }
  }

  return (
    <VStack
      align="start"
      padding="8"
      gap="4"
      bg="gray.50"
      borderRadius="md"
      borderWidth="thin"
      borderColor={borderColor}>
      <FormControl display="flex" gap="4">
        <OptionFormField
          id={`${FieldName.OPTIONS}.workspaceTemplate`}
          isRequired
          type="string"
          defaultValue={
            formState.defaultValues?.options?.workspaceTemplate ?? defaultTemplate?.metadata?.name
          }
          displayName="Infrastructure Template"
          enum={templates.map((template) => ({
            value: template.metadata!.name!,
            displayName: getDisplayName(template),
          }))}
          onChange={resetTemplate}
        />
        {currentTemplateVersions && currentTemplateVersions.length > 0 && (
          <OptionFormField
            id={`${FieldName.OPTIONS}.workspaceTemplateVersion`}
            type="string"
            defaultValue={formState.defaultValues?.options?.workspaceTemplateVersion ?? "latest"}
            displayName="Version"
            enum={[
              { value: "latest", displayName: "Latest" },
              ...currentTemplateVersions.map((version) => ({
                value: version.version,
                displayName: version.version,
              })),
            ]}
            onChange={resetTemplateVersion}
          />
        )}
      </FormControl>

      {currentParameters && currentParameters.length > 0 && (
        <SimpleGrid columns={[2]} gap="4" borderRadius={"md"} w="full">
          {currentParameters.map((param) => {
            const paramID = param.variable
            if (!paramID) {
              return null
            }
            const fieldID = `${FieldName.OPTIONS}.${paramID}`

            let defaultValue = param.defaultValue
            if (typeof formState.defaultValues?.options?.[paramID] === "string") {
              defaultValue = formState.defaultValues.options[paramID] as string
            }

            return (
              <OptionFormField
                key={fieldID}
                id={fieldID}
                displayName={param.label ?? paramID}
                description={param.description}
                defaultValue={defaultValue}
                type={convertParameterType(param.type)}
                enum={param.options?.map((option) => ({
                  value: option,
                  displayName: option,
                }))}
                isRequired={param.required}
              />
            )
          })}
        </SimpleGrid>
      )}
    </VStack>
  )
}

type TOptionFormFieldProps = Partial<
  Pick<TOptionWithID, "type" | "displayName" | "defaultValue" | "description" | "enum">
> &
  Readonly<{
    id: string
    isRequired?: boolean
    placeholder?: string
    onChange?: VoidFunction
  }>
function OptionFormField({
  id,
  defaultValue,
  description,
  type,
  displayName,
  enum: enumProp,
  placeholder,
  isRequired = false,
  onChange,
}: TOptionFormFieldProps) {
  const inputBackground = useColorModeValue("white", "black")
  const { register, formState, control } = useFormContext()
  const optionError = formState.errors[id]

  const input = useMemo<ReactNode>(() => {
    const registerProps = register(id, { required: isRequired })
    const defaultValueProp = exists(defaultValue) ? { defaultValue } : {}
    const props = {
      ...defaultValueProp,
      ...registerProps,
      onChange: (e: Parameters<ChangeHandler>[0]) => {
        registerProps.onChange(e)
        onChange?.()
      },
      background: inputBackground,
    }

    if (enumProp?.length) {
      let ph: string | undefined = placeholder ?? "Select option"
      if (defaultValue) {
        ph = undefined
      }

      return (
        <Select {...props} placeholder={ph}>
          {enumProp.map(
            (opt, i) =>
              opt.value && (
                <option key={i} value={opt.value}>
                  {opt.displayName ?? opt.value}
                </option>
              )
          )}
        </Select>
      )
    }

    switch (type) {
      case "boolean":
        return (
          <Controller
            name={id}
            control={control}
            rules={{ required: isRequired }}
            defaultValue={defaultValue}
            render={({ field: { onChange, onBlur, value } }) => {
              let isChecked = value
              if (typeof value === "string") {
                isChecked = value === "true"
              }

              return (
                <Switch
                  onChange={(e) => onChange(e.target.checked)}
                  onBlur={onBlur}
                  isChecked={isChecked}
                />
              )
            }}
          />
        )
      case "number":
        return (
          <Input
            spellCheck={false}
            placeholder={placeholder ?? `Enter ${displayName}`}
            type="number"
            {...props}
          />
        )
      case "duration":
        return (
          <Input
            spellCheck={false}
            placeholder={placeholder ?? `Enter ${displayName}`}
            type="text"
            {...props}
          />
        )
      case "string":
        return (
          <Input
            spellCheck={false}
            placeholder={placeholder ?? `Enter ${displayName}`}
            type="text"
            {...props}
          />
        )
      case "multiline":
        return (
          <Textarea
            rows={2}
            spellCheck={false}
            placeholder={placeholder ?? `Enter ${displayName}`}
            whiteSpace="pre"
            {...props}
          />
        )
      default:
        return (
          <Input
            spellCheck={false}
            placeholder={placeholder ?? `Enter ${displayName}`}
            type="text"
            {...props}
          />
        )
    }
  }, [
    register,
    id,
    isRequired,
    defaultValue,
    inputBackground,
    enumProp,
    type,
    onChange,
    placeholder,
    displayName,
    control,
  ])

  return (
    <FormControl isRequired={isRequired}>
      <FormLabel fontSize="sm">{displayName}</FormLabel>

      {exists(optionError) ? (
        <FormErrorMessage>{optionError.message?.toString() ?? "Error"}</FormErrorMessage>
      ) : (
        exists(description) && <FormHelperText userSelect="text">{description}</FormHelperText>
      )}

      {input}
    </FormControl>
  )
}

function convertParameterType(paramType: StorageV1AppParameter["type"]): TProviderOption["type"] {
  if (!paramType) {
    return undefined
  }

  switch (paramType) {
    case "string":
      return "string"
    case "multiline":
      return "multiline"
    case "number":
      return "number"
    case "password":
      return "string"
    case "boolean":
      return "boolean"
    default:
      return undefined
  }
}
