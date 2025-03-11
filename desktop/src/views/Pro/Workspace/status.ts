import { ProWorkspaceInstance } from "@/contexts"

export const InstancePhase = {
  Ready: "Ready",
  WaitingToInitialize: "",
  Sleeping: "Sleeping",
  Failed: "Failed",
  Deleting: "Deleting",
  Pending: "Pending",
} as const

export const InstanceStatus = {
  Running: "Running",
  Stopped: "Stopped",
  Busy: "Busy",
  NotFound: "NotFound",
} as const

export const WorkspaceDisplayStatus = {
  ...InstancePhase,
  ...InstanceStatus,
  Error: "Error",
} as const
export type TWorkspaceDisplayStatus =
  (typeof WorkspaceDisplayStatus)[keyof typeof WorkspaceDisplayStatus]

export function determineDisplayStatus(
  status: ProWorkspaceInstance["status"],
  deletionTimestamp: Date | undefined
): TWorkspaceDisplayStatus {
  const phase = status?.phase
  const lastWorkspaceStatus = status?.lastWorkspaceStatus
  if (deletionTimestamp) {
    return WorkspaceDisplayStatus.Deleting
  }

  if (!phase || phase === InstancePhase.Pending) {
    return WorkspaceDisplayStatus.Pending
  }

  if (phase === InstancePhase.Failed) {
    return WorkspaceDisplayStatus.Error
  }

  if (phase === InstancePhase.WaitingToInitialize) {
    return WorkspaceDisplayStatus.WaitingToInitialize
  }

  if (phase === InstancePhase.Ready) {
    if (lastWorkspaceStatus === InstanceStatus.NotFound) {
      return WorkspaceDisplayStatus.NotFound
    }

    if (lastWorkspaceStatus === InstanceStatus.Stopped) {
      return WorkspaceDisplayStatus.Stopped
    }

    if (lastWorkspaceStatus === InstanceStatus.Busy) {
      return WorkspaceDisplayStatus.Busy
    }

    if (lastWorkspaceStatus === InstanceStatus.Running) {
      return WorkspaceDisplayStatus.Running
    }

    return WorkspaceDisplayStatus.Ready
  }

  return phase as TWorkspaceDisplayStatus
}
