import { ReactElement } from "react"

export type TViewTitle = Readonly<{
  label: string
  priority: "high" | "regular"
  leadingAction?: ReactElement
  trailingAction?: ReactElement
}>
