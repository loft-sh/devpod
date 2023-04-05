import { useCallback, useEffect, useReducer } from "react"
import { client } from "../../../client"
import { useProviderManager } from "../../../contexts"
import { exists, TAction } from "../../../lib"
import { TProviderID, TProviderOptionGroup, TProviderOptions } from "../../../types"

type TSetupProviderState = Readonly<
  | {
      currentStep: 1
      providerID: null
      options: null
    }
  | {
      currentStep: 2
      providerID: TProviderID
      options: TProviderOptions
      optionGroups: TProviderOptionGroup[]
    }
  | { currentStep: "done"; providerID: TProviderID }
>
type TCompleteFirstStepAction = TAction<
  "completeFirstStep",
  Readonly<{
    providerID: TProviderID
    options: TProviderOptions
    optionGroups: TProviderOptionGroup[]
  }>
>
type TCompleteSecondStepAction = TAction<"completeSecondStep">
type TActions = TCompleteFirstStepAction | TCompleteSecondStepAction

const initialState: TSetupProviderState = { currentStep: 1, providerID: null, options: null }
function setupProviderReducer(state: TSetupProviderState, action: TActions): TSetupProviderState {
  switch (action.type) {
    case "completeFirstStep":
      return {
        ...state,
        currentStep: 2,
        providerID: action.payload.providerID,
        options: action.payload.options,
        optionGroups: action.payload.optionGroups,
      }
    case "completeSecondStep":
      return { currentStep: "done", providerID: state.providerID! }
    default:
      return state
  }
}

export function useSetupProvider() {
  const [state, dispatch] = useReducer(setupProviderReducer, initialState)

  const completeFirstStep = useCallback(
    (payload: TCompleteFirstStepAction["payload"]) => {
      if (state.currentStep !== 1) {
        return
      }

      dispatch({ type: "completeFirstStep", payload })
    },
    [state.currentStep]
  )

  const completeSecondStep = useCallback(() => {
    if (state.currentStep !== 2) {
      return
    }

    dispatch({ type: "completeSecondStep" })
  }, [state.currentStep])

  const { remove } = useProviderManager()
  useEffect(() => {
    if (state.currentStep === 1 || state.currentStep === "done") {
      client.providers.popDangling()

      return
    }

    client.providers.setDangling(state.providerID)
  }, [state])

  useEffect(() => {
    return () => {
      const danglingProviderID = client.providers.popDangling()
      if (exists(danglingProviderID)) {
        remove.run({ providerID: danglingProviderID })
      }
    }
    // We need to ensure this effect only runs when the hook unmounts at the cost of potentially stale dependencies
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return { state, completeFirstStep, completeSecondStep }
}
