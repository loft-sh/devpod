import { useFormContext } from "react-hook-form"
import { FieldName, TFormValues } from "@/views/Pro/CreateWorkspace/types"
import { Select, useToken } from "@chakra-ui/react"

export function SourceSelect({ resetPreset }: { resetPreset: VoidFunction }) {
  const errorBorderColor = useToken("colors", "red.500")
  const { register, trigger: validate, getFieldState } = useFormContext<TFormValues>()

  const sourceState = getFieldState(FieldName.SOURCE)
  const sourceError = sourceState.isDirty && sourceState.invalid

  return (
    <Select
      {...register(FieldName.SOURCE_TYPE, {
        onChange: () => {
          validate(FieldName.SOURCE)
          resetPreset()
        },
      })}
      isInvalid={sourceError}
      aria-invalid={sourceError ? "true" : undefined}
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
      <option value="git">Repo</option>
      <option value="local">Local Folder</option>
      <option value="image">Image</option>
    </Select>
  )
}
