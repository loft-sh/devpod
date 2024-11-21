import { useFormContext } from "react-hook-form"
import { FieldName, TFormValues } from "@/views/Pro/CreateWorkspace/types"
import { Select } from "@chakra-ui/react"
import { ManagementV1Runner } from "@loft-enterprise/client/gen/models/managementV1Runner"

export function RunnerInput({ runners }: { runners: readonly ManagementV1Runner[] | undefined }) {
  const { register } = useFormContext<TFormValues>()

  return (
    <Select {...register(FieldName.RUNNER)}>
      {runners?.map((r, index) => (
        <option key={index} value={r.metadata?.name}>
          {r.spec?.displayName ?? r.metadata?.name}
        </option>
      ))}
    </Select>
  )
}
