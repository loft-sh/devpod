import { TIDE, TIdentifiable, TWorkspaceSource } from "@/types"
import { ManagementV1DevPodWorkspaceInstance } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceInstance"
import { Labels, deepCopy } from "@/lib"
import { Resources } from "@loft-enterprise/client"
import { ManagementV1DevPodWorkspaceInstanceStatus } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceInstanceStatus"

export class ProWorkspaceInstance
  extends ManagementV1DevPodWorkspaceInstance
  implements TIdentifiable
{
  public readonly status: ProWorkspaceInstanceStatus | undefined

  public get id(): string {
    const maybeID = this.metadata?.labels?.[Labels.WorkspaceID]
    if (!maybeID) {
      // If we don't have an ID we should ignore the instance.
      // Throwing an error for now to see how often this happens
      throw new Error(`No Workspace ID label present on instance ${this.metadata?.name}`)
    }

    return maybeID
  }

  constructor(instance: ManagementV1DevPodWorkspaceInstance) {
    super()

    this.apiVersion = `${Resources.ManagementV1DevPodWorkspaceInstance.group}/${Resources.ManagementV1DevPodWorkspaceInstance.version}`
    this.kind = Resources.ManagementV1DevPodWorkspaceInstance.kind
    this.metadata = deepCopy(instance.metadata)
    this.spec = deepCopy(instance.spec)
    this.status = deepCopy(instance.status) as ProWorkspaceInstanceStatus
  }
}

class ProWorkspaceInstanceStatus extends ManagementV1DevPodWorkspaceInstanceStatus {
  "source"?: TWorkspaceSource
  "ide"?: TIDE
  "metrics"?: ProWorkspaceMetricsSummary

  constructor() {
    super()
  }
}

class ProWorkspaceMetricsSummary {
  "latencyMs"?: number
  "connectionType"?: "direct" | "DERP"
  "derpRegion"?: string
}
