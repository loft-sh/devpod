import { QueryKeys } from "@/queryKeys"
import { useQuery } from "@tanstack/react-query"
import { useProContext } from "@/contexts"

type TConnectionState = "connected" | "disconnected"
type TConnectionStatus = {
  state?: TConnectionState
  isLoading: boolean
  details?: {
    platform: {
      state: TConnectionState
    }
    daemon?: {
      state: TConnectionState
    }
  }
}
export function useConnectionStatus(): TConnectionStatus {
  const { host, client } = useProContext()
  const { data: connection, isLoading } = useQuery({
    queryKey: QueryKeys.connectionStatus(host),
    queryFn: async () => {
      try {
        const connectionStatus: Omit<TConnectionStatus, "isLoading"> = {
          state: "disconnected",
          details: {
            platform: {
              state: "disconnected",
            },
          },
        }

        const platformRes = await client.checkPlatformHealth()
        if (platformRes.ok && platformRes.val.healthy) {
          connectionStatus.details!.platform.state = "connected"
          connectionStatus.state = "connected"
        }

        const daemonRes = await client.checkDaemonHealth()
        if (daemonRes.ok && daemonRes.val.found && daemonRes.val.status?.state == "running") {
          connectionStatus.details!.daemon = {
            state: "connected",
          }
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
