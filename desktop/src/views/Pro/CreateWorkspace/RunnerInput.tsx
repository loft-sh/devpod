import { useFormContext } from "react-hook-form"
import { FieldName, TFormValues } from "@/views/Pro/CreateWorkspace/types"
import { Select } from "@chakra-ui/react"
import { ManagementV1ProjectClusters } from "@loft-enterprise/client/gen/models/managementV1ProjectClusters"

export function TargetInput({
  projectClusters,
}: {
  projectClusters: ManagementV1ProjectClusters | undefined
}) {
  const { register } = useFormContext<TFormValues>()

  return (
    <Select {...register(FieldName.TARGET)}>
      {projectClusters?.clusters?.map((r, index) => (
        <option key={index} value={r.metadata?.name}>
          {r.spec?.displayName ?? r.metadata?.name}
        </option>
      ))}
    </Select>
  )
}
