import { ManagementV1DevPodWorkspacePreset } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspacePreset"

export function presetDisplayName(preset: ManagementV1DevPodWorkspacePreset | undefined) {
  return preset?.spec?.displayName ?? preset?.metadata?.name
}
