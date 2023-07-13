import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useMemo } from "react"
import { client } from "../../client"
import { QueryKeys } from "../../queryKeys"
import { TIDE } from "../../types"

export function useIDESettings() {
  const queryClient = useQueryClient()
  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })
  const { mutate: updateDefaultIDE } = useMutation({
    mutationFn: async ({ ide }: { ide: NonNullable<TIDE["name"]> }) => {
      ;(await client.ides.useIDE(ide)).unwrap()
    },
    onSettled: () => {
      queryClient.invalidateQueries(QueryKeys.IDES)
    },
  })

  return useMemo(
    () => ({
      ides: idesQuery.data ?? [],
      defaultIDE: idesQuery.data?.find((ide) => ide.default),
      updateDefaultIDE,
    }),
    [idesQuery.data, updateDefaultIDE]
  )
}
