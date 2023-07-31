import { useQuery } from "@tanstack/react-query"
import { client } from "../client"
import { Release } from "../gen"
import { QueryKeys } from "../queryKeys"

export function useReleases(): readonly Release[] | undefined {
  const { data: releases } = useQuery({
    queryKey: QueryKeys.RELEASES,
    queryFn: async () => {
      return (await client.fetchReleases()).unwrap()
    },
  })

  return releases
}
