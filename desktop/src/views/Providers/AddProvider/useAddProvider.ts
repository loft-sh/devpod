import { useCallback, useReducer } from "react"
import { TAction } from "../../../lib"
import { TProviderID, TProviderOptions } from "../../../types"

type TAddProviderState = Readonly<
  | {
      currentStep: 1
      providerID: null
      options: null
    }
  | {
      currentStep: 2
      providerID: TProviderID
      options: TProviderOptions
    }
  | { currentStep: "done"; providerID: TProviderID }
>
type TCompleteFirstStepAction = TAction<
  "completeFirstStep",
  Readonly<{ providerID: TProviderID; options: TProviderOptions }>
>
type TCompleteSecondStepAction = TAction<"completeSecondStep">

type TActions = TCompleteFirstStepAction | TCompleteSecondStepAction

const initialState: TAddProviderState = { currentStep: 1, providerID: null, options: null }
function addProviderReducer(state: TAddProviderState, action: TActions): TAddProviderState {
  switch (action.type) {
    case "completeFirstStep":
      return {
        ...state,
        currentStep: 2,
        providerID: action.payload.providerID,
        options: action.payload.options,
      }
    case "completeSecondStep":
      return { currentStep: "done", providerID: state.providerID! }
    default:
      return state
  }
}

export function useAddProvider() {
  const [state, dispatch] = useReducer(addProviderReducer, initialState)

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

  return { state, completeFirstStep, completeSecondStep }
}
