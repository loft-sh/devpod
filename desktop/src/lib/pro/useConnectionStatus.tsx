import { QueryKeys } from "@/queryKeys"
import { useQuery } from "@tanstack/react-query"
import { useProContext } from "@/contexts"

type TConnectionStatus = Readonly<{
  state?: "connected" | "disconnected"
  isLoading: boolean
}>
export function useConnectionStatus(): TConnectionStatus {
  const { host, client } = useProContext()
  const { data: connection, isLoading } = useQuery({
    queryKey: QueryKeys.connectionStatus(host),
    queryFn: async () => {
      try {
        const res = await client.checkHealth()
        let state: TConnectionStatus["state"] = "disconnected"
        if (res.err) {
          return { state }
        }

        if (res.val.healthy) {
          state = "connected"
        }

        return { state }
      } catch {
        return { state: "disconnected" as TConnectionStatus["state"] }
      }
    },
    refetchInterval: 5_000,
  })

  return { ...connection, isLoading }
}
