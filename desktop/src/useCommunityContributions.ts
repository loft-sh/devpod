import { useQuery } from "@tanstack/react-query"
import { client } from "./client"
import { TCommunityContributions } from "./types"
import { QueryKeys } from "./queryKeys"

export function useCommunityContributions(): Readonly<{
  contributions: TCommunityContributions | undefined
  isLoading: boolean
}> {
  const { data, isLoading } = useQuery({
    queryKey: QueryKeys.COMMUNITY_CONTRIBUTIONS,
    queryFn: async () => {
      return (await client.fetchCommunityContributions()).unwrap()
    },
    refetchOnWindowFocus: false,
    refetchOnMount: false,
    refetchOnReconnect: false,
  })

  return { contributions: data, isLoading }
}
