import { useContext } from "react"
import { TProviderManager } from "../../types"
import { DevPodContext, TDevpodContext } from "./DevPodProvider"
import { useProviderManager } from "./useProviderManager"

export function useProviders(): [TDevpodContext["providers"], TProviderManager] {
  const providers = useContext(DevPodContext).providers
  const manager = useProviderManager()

  return [providers, manager]
}
