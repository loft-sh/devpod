import { TProviderID } from "../../types"

export const FieldName = {
  SOURCE: "source",
  ID: "id",
  DEFAULT_IDE: "defaultIDE",
  PROVIDER: "provider",
} as const

export type TFormValues = {
  [FieldName.SOURCE]: string
  [FieldName.DEFAULT_IDE]: string
  [FieldName.PROVIDER]: TProviderID // TODO: needs runtime validation
  [FieldName.ID]: string
}
