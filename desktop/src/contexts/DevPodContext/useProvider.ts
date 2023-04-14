import { TProvider, TProviderID, TProviderManager } from "../../types"
import { useProviders } from "./useProviders"

export function useProvider(
  providerID: TProviderID | undefined | null
): [TProvider | undefined, TProviderManager] {
  const [[providers], manager] = useProviders()

  return [providerID ? providers?.[providerID] : undefined, manager]
}
