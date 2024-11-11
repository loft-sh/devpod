import { ProWorkspaceInstance } from "@/contexts"
import { TWorkspaceResult } from "@/contexts/DevPodContext/workspaces/useWorkspace"
import { ManagementV1DevPodWorkspaceTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceTemplate"

export type TTabProps = Readonly<{
  host: string
  workspace: TWorkspaceResult<ProWorkspaceInstance>
  instance: ProWorkspaceInstance
  template: ManagementV1DevPodWorkspaceTemplate | undefined
}>
