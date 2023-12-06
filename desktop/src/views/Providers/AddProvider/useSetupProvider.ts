import { useCallback, useEffect, useReducer } from "react"
import { client } from "../../../client"
import { useProviderManager } from "../../../contexts"
import { TAction } from "../../../lib"
import { TProviderID } from "../../../types"

export type TSetupProviderState = Readonly<
  | {
      currentStep: "select-provider"
      providerID: null
      suggestedOptions: null
    }
  | {
      currentStep: "configure-provider"
      providerID: TProviderID
      suggestedOptions: Record<string, string>
    }
  | { currentStep: "done"; providerID: TProviderID }
>
type TCompleteSetupProviderAction = TAction<
  "completeSetupProvider",
  Readonly<{
    providerID: TProviderID
    suggestedOptions: Record<string, string>
  }>
>

type TActions =
  | TCompleteSetupProviderAction
  | TAction<"completeConfigureProvider">
  | TAction<"reset">

const initialState: TSetupProviderState = {
  currentStep: "select-provider",
  providerID: null,
  suggestedOptions: null,
}
function setupProviderReducer(state: TSetupProviderState, action: TActions): TSetupProviderState {
  switch (action.type) {
    case "reset":
      return initialState
    case "completeSetupProvider":
      return {
        ...state,
        currentStep: "configure-provider",
        providerID: action.payload.providerID,
        suggestedOptions: action.payload.suggestedOptions,
      }
    case "completeConfigureProvider":
      return { currentStep: "done", providerID: state.providerID! }
    default:
      return state
  }
}

export function useSetupProvider() {
  const [state, dispatch] = useReducer(setupProviderReducer, initialState)
  const { remove } = useProviderManager()

  const reset = useCallback(() => {
    dispatch({ type: "reset" })
  }, [])

  const completeSetupProvider = useCallback((payload: TCompleteSetupProviderAction["payload"]) => {
    dispatch({ type: "completeSetupProvider", payload })
  }, [])

  const completeConfigureProvider = useCallback(() => {
    if (state.currentStep !== "configure-provider") {
      return
    }

    dispatch({ type: "completeConfigureProvider" })
  }, [state.currentStep])

  const removeDanglingProviders = useCallback(() => {
    const danglingProviderIDs = client.providers.popAllDangling()
    for (const danglingProviderID of danglingProviderIDs) {
      remove.run({ providerID: danglingProviderID })
    }
  }, [remove])

  useEffect(() => {
    if (state.currentStep === "done") {
      client.providers.popDangling()

      return
    }
    if (state.providerID === null) {
      return
    }

    client.providers.setDangling(state.providerID)
  }, [state])

  useEffect(() => {
    return () => {
      removeDanglingProviders()
    }
    // We need to ensure this effect only runs when the hook unmounts at the cost of potentially stale dependencies
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return { state, reset, completeSetupProvider, completeConfigureProvider, removeDanglingProviders }
}
