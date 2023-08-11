import { useQuery } from "@tanstack/react-query"
import { client } from "../client"
import { QueryKeys } from "../queryKeys"

export function useVersion(): string | undefined {
  const { data: version } = useQuery({
    queryKey: QueryKeys.APP_VERSION,
    queryFn: () => client.fetchVersion(),
    cacheTime: Infinity,
    staleTime: Infinity,
  })

  return version
}
