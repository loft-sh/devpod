import { getDisplayName } from "@/lib"
import { Input, InputGroup, InputLeftAddon, Select, useToken } from "@chakra-ui/react"
import { ManagementV1DevPodEnvironmentTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodEnvironmentTemplate"
import { useMemo } from "react"
import { useFormContext } from "react-hook-form"
import { FieldName } from "./types"

type TDevContainerInputProps = Readonly<{
  environmentTemplates: readonly ManagementV1DevPodEnvironmentTemplate[]
}>
export function DevContainerInput({ environmentTemplates: templates }: TDevContainerInputProps) {
  const errorBorderColor = useToken("colors", "red.500")
  const { register, watch, resetField } = useFormContext()
  const devContainerType = watch(FieldName.DEVCONTAINER_TYPE, "path")

  const inputProps = useMemo(
    () => register(FieldName.DEVCONTAINER_JSON, { required: false }),
    [register]
  )

  const { input } = useMemo(() => {
    if (devContainerType === "path") {
      return { input: <Input {...inputProps} placeholder="path/to/devcontainer.json" /> }
    }

    return {
      input: (
        <Select
          w="full"
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
    <InputGroup bg="white">
      <InputLeftAddon padding="0">
        <Select
          {...register(FieldName.DEVCONTAINER_TYPE, {
            onChange: () => resetField(FieldName.DEVCONTAINER_JSON),
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
  )
}
