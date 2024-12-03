import { TWorkspaceSourceType } from "@/types"

export const FieldName = {
  SOURCE: "source",
  SOURCE_TYPE: "sourceType",
  NAME: "name",
  DEFAULT_IDE: "defaultIDE",
  RUNNER: "runner",
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
  [FieldName.RUNNER]: string
  [FieldName.ENV_TEMPLATE_VERSION]: string
  [FieldName.DEVCONTAINER_TYPE]: TDevContainerType
  [FieldName.OPTIONS]: TOptions
}

type TOptions = {
  workspaceTemplate?: string
  workspaceTemplateVersion?: string
  [key: string]: string | Record<string, unknown> | undefined
}

export type TDevContainerType = "path" | "external"
