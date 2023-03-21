import { useQuery } from "@tanstack/react-query"
import { client, TArch, TPlatform } from "../client"
import { QueryKeys } from "../queryKeys"

export function usePlatform(): TPlatform | undefined {
  const { data: platform } = useQuery({
    queryKey: QueryKeys.PLATFORM,
    queryFn: () => client.fetchPlatform(),
  })

  return platform
}

export function useArch(): TArch | undefined {
  const { data: arch } = useQuery({
    queryKey: QueryKeys.ARCHITECTURE,
    queryFn: () => client.fetchArch(),
  })

  return arch
}
