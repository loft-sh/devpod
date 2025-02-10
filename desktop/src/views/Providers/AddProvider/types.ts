import { TProvider, TProviderID, TWithProviderID } from "../../../types"

export const FieldName = {
  PROVIDER_SOURCE: "providerSource",
  PROVIDER_NAME: "providerName",
} as const
export type TFormValues = {
  [FieldName.PROVIDER_SOURCE]: string
  [FieldName.PROVIDER_NAME]: string | undefined
}

export type TCloneProviderInfo = Readonly<{
  sourceProviderID: TProviderID
  sourceProvider: TProvider
  sourceProviderSource: NonNullable<NonNullable<NonNullable<TProvider["config"]>["source"]>["raw"]>
}>

export type TSetupProviderResult = TWithProviderID &
  Readonly<{ suggestedOptions: Record<string, string> }>
