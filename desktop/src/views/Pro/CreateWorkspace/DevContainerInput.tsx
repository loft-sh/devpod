import { getDisplayName } from "@/lib"
import { Box, Input, InputGroup, InputLeftAddon, Select, useToken } from "@chakra-ui/react"
import { ManagementV1DevPodEnvironmentTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodEnvironmentTemplate"
import { useEffect, useMemo, useState } from "react"
import { useFormContext } from "react-hook-form"
import { FieldName } from "./types"

type TDevContainerInputProps = Readonly<{
  environmentTemplates: readonly ManagementV1DevPodEnvironmentTemplate[]
  resetPreset?: VoidFunction
}>
export function DevContainerInput({
  resetPreset,
  environmentTemplates: templates,
}: TDevContainerInputProps) {
  const errorBorderColor = useToken("colors", "red.500")
  const { register, watch, resetField } = useFormContext()
  const devContainerType = watch(FieldName.DEVCONTAINER_TYPE, "path")
  const envTemplateValue = watch(FieldName.DEVCONTAINER_JSON)

  const [envReference, setEnvReference] = useState<string | undefined>(envTemplateValue)

  // Need the extra render cycle, because the form will not report the default value of the
  // environment template immediately if the type is changed.
  useEffect(() => {
    setEnvReference(envTemplateValue)
  }, [envTemplateValue])

  const envTemplate = useMemo(() => {
    return determineEnvironmentTemplate(templates, envReference, devContainerType)
  }, [templates, envReference, devContainerType])

  const versions = useMemo(() => {
    if (devContainerType === "path") {
      return undefined
    }

    return envTemplate?.spec?.versions?.map((v) => v.version!)
  }, [envTemplate, devContainerType])

  const inputProps = useMemo(
    () =>
      register(FieldName.DEVCONTAINER_JSON, {
        required: false,
        onChange: () => {
          resetPreset?.()
          resetField(FieldName.ENV_TEMPLATE_VERSION, { defaultValue: "latest" })
        },
      }),
    [register, resetField, resetPreset]
  )

  const { input } = useMemo(() => {
    if (devContainerType === "path") {
      return { input: <Input {...inputProps} placeholder="path/to/devcontainer.json" /> }
    }

    return {
      input: (
        <Select
          w="full"
          cursor={"pointer"}
          {...inputProps}
          isDisabled={templates.length === 0}
          placeholder={templates.length === 0 ? "No templates available" : ""}>
          {templates.map((template) => (
            <option key={template.metadata!.name} value={template.metadata!.name}>
              {getDisplayName(template)}
            </option>
          ))}
        </Select>
      ),
    }
  }, [devContainerType, inputProps, templates])

  return (
    <Box display={"flex"} flexDirection={"row"} w={"full"} gap={"4"}>
      <Box flexGrow={1}>
        <InputGroup bg="white" w={"full"}>
          <InputLeftAddon padding="0">
            <Select
              {...register(FieldName.DEVCONTAINER_TYPE, {
                onChange: () => {
                  resetPreset?.()
                  resetField(FieldName.ENV_TEMPLATE_VERSION, { defaultValue: "latest" })
                  resetField(FieldName.DEVCONTAINER_JSON)
                },
              })}
              _invalid={{
                borderStyle: "solid",
                borderWidth: "1px",
                borderRightWidth: 0,
                borderColor: errorBorderColor,
              }}
              borderTopRightRadius="0"
              borderBottomRightRadius="0"
              focusBorderColor="transparent"
              cursor="pointer"
              w="full"
              border="none">
              <option value="path">Path</option>
              <option value="external">External</option>
            </Select>
          </InputLeftAddon>
          {input}
        </InputGroup>
      </Box>

      {versions?.length && (
        <Box w={48}>
          <InputGroup bg={"white"} w={"full"}>
            <Select
              {...register(FieldName.ENV_TEMPLATE_VERSION, {
                onChange: () => {
                  resetPreset?.()
                },
              })}
              _invalid={{
                borderStyle: "solid",
                borderWidth: "1px",
                borderRightWidth: 0,
                borderColor: errorBorderColor,
              }}
              cursor="pointer"
              w="full">
              <option value={"latest"}>Latest</option>
              {versions.map((v) => (
                <option key={v} value={v}>
                  {v}
                </option>
              ))}
            </Select>
          </InputGroup>
        </Box>
      )}
    </Box>
  )
}

function determineEnvironmentTemplate(
  templates: readonly ManagementV1DevPodEnvironmentTemplate[],
  envTemplateValue: string | undefined,
  devContainerType: "path" | "external" | undefined
): ManagementV1DevPodEnvironmentTemplate | undefined {
  if (devContainerType === "path" || !envTemplateValue) {
    return undefined
  }

  return templates.find((t) => t.metadata?.name === envTemplateValue)
}
