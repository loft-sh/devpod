import { useProContext } from "@/contexts"
import { useQuery, UseQueryResult } from "@tanstack/react-query"
import { QueryKeys } from "@/queryKeys"
import { ManagementV1DevPodWorkspaceTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceTemplate"
import { ManagementV1DevPodEnvironmentTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodEnvironmentTemplate"

type TTemplates = Readonly<{
  default: ManagementV1DevPodWorkspaceTemplate | undefined
  workspace: readonly ManagementV1DevPodWorkspaceTemplate[]
  environment: readonly ManagementV1DevPodEnvironmentTemplate[]
}>
export function useTemplates(): UseQueryResult<TTemplates> {
  const { host, currentProject, client } = useProContext()
  const query = useQuery<TTemplates>({
    // eslint-disable-next-line @tanstack/query/exhaustive-deps
    queryKey: QueryKeys.proWorkspaceTemplates(host, currentProject?.metadata!.name!),
    queryFn: async () => {
      const projectTemplates = (
        await client.getProjectTemplates(currentProject?.metadata!.name!)
      ).unwrap()

      // try to find default template in list
      let defaultTemplate: ManagementV1DevPodWorkspaceTemplate | undefined = undefined
      if (projectTemplates?.defaultDevPodWorkspaceTemplate) {
        defaultTemplate = projectTemplates.devPodWorkspaceTemplates?.find(
          (template) => template.metadata?.name === projectTemplates.defaultDevPodWorkspaceTemplate
        )
      }

      return {
        default: defaultTemplate,
        workspace: projectTemplates?.devPodWorkspaceTemplates ?? [],
        environment: projectTemplates?.devPodEnvironmentTemplates ?? [],
      }
    },
    enabled: !!currentProject,
  })

  return query
}
