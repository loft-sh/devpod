import { useProContext } from "@/contexts"
import { QueryKeys } from "@/queryKeys"
import { TPlatformHealthCheck } from "@/types"
import { useQuery } from "@tanstack/react-query"

export type TConnectionStatus = Partial<TPlatformHealthCheck> & {
  isLoading: boolean
}
export function useConnectionStatus(): TConnectionStatus {
  const { host, client } = useProContext()
  const { data: connection, isLoading } = useQuery({
    queryKey: QueryKeys.connectionStatus(host),
    queryFn: async () => {
      try {
        const platformRes = await client.checkHealth()
        if (platformRes.ok) {
          return platformRes.val
        }

        return { healthy: false }
      } catch {
        return { healthy: false }
      }
    },
    refetchInterval: 5_000,
  })

  return { ...connection, isLoading }
}
