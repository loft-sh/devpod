import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useMemo } from "react"
import { client } from "../../client"
import { QueryKeys } from "../../queryKeys"
import { TIDE } from "../../types"
import { useIDEs } from "../../useIDEs"

export function useIDESettings() {
  const queryClient = useQueryClient()
  const { ides, defaultIDE } = useIDEs()
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
      ides,
      defaultIDE,
      updateDefaultIDE,
    }),
    [defaultIDE, ides, updateDefaultIDE]
  )
}
