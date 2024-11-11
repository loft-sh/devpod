import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useCallback, useState } from "react"
import { client } from "../client"
import { QueryKeys } from "../queryKeys"

export function useUpdate() {
  const queryClient = useQueryClient()
  const {
    data: isUpdateAvailable,
    refetch,
    // Need to use `isFetching` as a workaround for interaction with `enabled: false`
    isFetching: isChecking,
  } = useQuery({
    queryKey: QueryKeys.UPDATE_RELEASE,
    queryFn: async () => (await client.checkUpdates()).unwrap(),
    enabled: false,
    onSettled: () => {
      queryClient.resetQueries(QueryKeys.PENDING_UPDATE)
    },
  })

  const { data: pendingUpdate } = useQuery({
    queryKey: QueryKeys.PENDING_UPDATE,
    queryFn: async () => (await client.fetchPendingUpdate()).unwrap(),
    enabled: false,
  })

  const { mutate: installMutate, isLoading: isInstalling } = useMutation({
    mutationKey: QueryKeys.INSTALL_UPDATE,
    mutationFn: async () => (await client.installUpdate()).unwrap(),
    onSettled: () => {
      queryClient.resetQueries(QueryKeys.PENDING_UPDATE)
    },
  })

  const [isInstallDisabled, setIsInstallDisabled] = useState(false)
  const install = useCallback(() => {
    if (isInstallDisabled) {
      return
    }

    installMutate(undefined, {
      onError: () => setIsInstallDisabled(false),
      onSuccess: () => setIsInstallDisabled(true),
    })
  }, [installMutate, isInstallDisabled])

  return {
    check: refetch,
    isUpdateAvailable,
    isChecking,
    pendingUpdate,
    install,
    isInstalling,
    isInstallDisabled,
  }
}
