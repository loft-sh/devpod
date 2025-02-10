import { useContext } from "react"
import { TProviderManager } from "../../types"
import { DevPodContext, TDevpodContext } from "./DevPodProvider"
import { useProviderManager } from "./useProviderManager"

export function useProviders(): [TDevpodContext["providers"] | [undefined], TProviderManager] {
  const providers = useContext(DevPodContext)?.providers ?? [undefined]
  const manager = useProviderManager()

  return [providers, manager]
}
