import { useQuery } from "@tanstack/react-query"
import { client } from "../client"
import { QueryKeys } from "../queryKeys"

export function usePlatform(): TPlatform | undefined {
  const { data: platform } = useQuery([QueryKeys.PLATFORM], () => client.fetchPlatform())

  return platform
}

export function useArch(): TArch | undefined {
  const { data: arch } = useQuery([QueryKeys.ARCHITECTURE], () => client.fetchArch())

  return arch
}
