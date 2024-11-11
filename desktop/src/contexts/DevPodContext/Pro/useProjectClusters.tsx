import { useProContext } from "@/contexts"
import { QueryKeys } from "@/queryKeys"
import { ManagementV1Cluster } from "@loft-enterprise/client/gen/models/managementV1Cluster"
import { ManagementV1Runner } from "@loft-enterprise/client/gen/models/managementV1Runner"
import { useQuery, UseQueryResult } from "@tanstack/react-query"

type TProjectClusters = Readonly<{
  clusters: readonly ManagementV1Cluster[]
  runners: readonly ManagementV1Runner[]
}>

export function useProjectClusters(): UseQueryResult<TProjectClusters> {
  const { host, currentProject, client } = useProContext()
  const query = useQuery<TProjectClusters>({
    // eslint-disable-next-line @tanstack/query/exhaustive-deps
    queryKey: QueryKeys.proClusters(host, currentProject?.metadata!.name!),
    queryFn: async () => {
      const projectClusters = (
        await client.getProjectClusters(currentProject?.metadata!.name!)
      ).unwrap()

      return {
        runners: projectClusters?.runners ?? [],
        clusters: projectClusters?.clusters ?? [],
      }
    },
    enabled: !!currentProject,
  })

  return query
}
