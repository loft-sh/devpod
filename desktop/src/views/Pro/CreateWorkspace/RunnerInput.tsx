import { useFormContext } from "react-hook-form"
import { FieldName, TFormValues } from "@/views/Pro/CreateWorkspace/types"
import { Select } from "@chakra-ui/react"
import { TProjectCluster } from "@/contexts/DevPodContext/Pro/useProjectClusters"

export function TargetInput({
  projectClusters,
}: {
  projectClusters: TProjectCluster | undefined
}) {
  const { register } = useFormContext<TFormValues>()

  const clusters =
    projectClusters?.runners && projectClusters.runners.length > 0
      ? projectClusters.runners
      : projectClusters?.clusters

  return (
    <Select {...register(FieldName.TARGET)}>
      {clusters?.map((r, index) => (
        <option key={index} value={r.metadata?.name}>
          {r.spec?.displayName ?? r.metadata?.name}
        </option>
      ))}
    </Select>
  )
}
