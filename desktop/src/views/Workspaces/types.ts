import { TProviderID } from "../../types"

export const SUPPORTED_IDES = ["vscode", "intellj"] as const
type TSupportedIDE = (typeof SUPPORTED_IDES)[number]

export const FieldName = {
  SOURCE: "source",
  ID: "id",
  DEFAULT_IDE: "defaultIDE",
  PROVIDER: "provider",
} as const

export type TFormValues = {
  [FieldName.SOURCE]: string
  [FieldName.DEFAULT_IDE]: TSupportedIDE
  [FieldName.PROVIDER]: TProviderID // TODO: needs runtime validation
  [FieldName.ID]: string
}
