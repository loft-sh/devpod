import { TWorkspaceSourceType } from "@/types"

export const FieldName = {
  SOURCE: "source",
  SOURCE_TYPE: "sourceType",
  NAME: "name",
  DEFAULT_IDE: "defaultIDE",
  TARGET: "target",
  DEVCONTAINER_JSON: "devcontainerJSON",
  ENV_TEMPLATE_VERSION: "envTemplateVersion",
  DEVCONTAINER_TYPE: "devcontainerType",
  OPTIONS: "options",
} as const

export type TFormValues = {
  [FieldName.SOURCE]: string
  [FieldName.SOURCE_TYPE]: TWorkspaceSourceType
  [FieldName.DEFAULT_IDE]: string
  [FieldName.NAME]: string
  [FieldName.DEVCONTAINER_JSON]: string
  [FieldName.TARGET]: TTarget
  [FieldName.ENV_TEMPLATE_VERSION]: string
  [FieldName.DEVCONTAINER_TYPE]: TDevContainerType
  [FieldName.OPTIONS]: TOptions
}

type TTarget = string // either runner or cluster, depending on provider
type TOptions = {
  workspaceTemplate?: string
  workspaceTemplateVersion?: string
  [key: string]: string | boolean | number | Record<string, unknown> | undefined
}

export type TDevContainerType = "path" | "external"
