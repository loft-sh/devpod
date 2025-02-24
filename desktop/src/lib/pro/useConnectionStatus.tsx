import { QueryKeys } from "@/queryKeys"
import { useQuery } from "@tanstack/react-query"
import { useProContext } from "@/contexts"

type TConnectionState = "connected" | "disconnected"
type TConnectionStatus = {
  state?: TConnectionState
  isLoading: boolean
}
export function useConnectionStatus(): TConnectionStatus {
  const { host, client } = useProContext()
  const { data: connection, isLoading } = useQuery({
    queryKey: QueryKeys.connectionStatus(host),
    queryFn: async () => {
      try {
        const connectionStatus: Omit<TConnectionStatus, "isLoading"> = {
          state: "disconnected",
        }

        const platformRes = await client.checkHealth()
        if (platformRes.ok && platformRes.val.healthy) {
          connectionStatus.state = "connected"
        }

        return connectionStatus
      } catch {
        return { state: "disconnected" as TConnectionStatus["state"] }
      }
    },
    refetchInterval: 5_000,
  })

  return { ...connection, isLoading }
}
