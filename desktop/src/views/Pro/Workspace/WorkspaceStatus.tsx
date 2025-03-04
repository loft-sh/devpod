import { ProWorkspaceInstance } from "@/contexts"
import { CheckCircle, CircleDuotone, Clock, ExclamationTriangle, NotFound, Sleep } from "@/icons"
import { BoxProps, HStack, Text } from "@chakra-ui/react"
import { V1ObjectMeta } from "@loft-enterprise/client/gen/models/V1ObjectMeta"
import React from "react"

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

const badgeOptionMappings: {
  [key in TWorkspaceDisplayStatus]?: Pick<TStatusBadgeProps, "icon" | "color">
} = {
  [WorkspaceDisplayStatus.Pending]: {
    icon: <Clock boxSize={5} />,
    color: "orange.500",
  },
  [WorkspaceDisplayStatus.Sleeping]: {
    icon: <Sleep boxSize={5} />,
    color: "#706BFF",
  },
  [WorkspaceDisplayStatus.Error]: {
    icon: <ExclamationTriangle boxSize={5} />,
    color: "red.500",
  },
  [WorkspaceDisplayStatus.NotFound]: {
    icon: <NotFound boxSize={5} />,
    color: "gray.600",
  },
  [WorkspaceDisplayStatus.Stopped]: {
    icon: <CircleDuotone boxSize={5} />,
    color: "red.400",
  },
  [WorkspaceDisplayStatus.Busy]: {
    icon: <CircleDuotone boxSize={5} />,
    color: "red.500",
  },
  [WorkspaceDisplayStatus.Failed]: {
    icon: <CircleDuotone boxSize={5} />,
    color: "red.500",
  },
  [WorkspaceDisplayStatus.Deleting]: {
    icon: <CircleDuotone boxSize={5} />,
    color: "red.500",
  },
  [WorkspaceDisplayStatus.Running]: {
    icon: <CircleDuotone boxSize={5} />,
    color: "primary.500",
  },
  [WorkspaceDisplayStatus.Ready]: {
    icon: <CheckCircle boxSize={5} />,
    color: "primary.400",
  },
  [WorkspaceDisplayStatus.WaitingToInitialize]: {
    icon: <CircleDuotone boxSize={5} />,
    color: "gray.600",
  },
}

type TWorkspaceStatusProps = Readonly<{
  status: ProWorkspaceInstance["status"]
  deletionTimestamp: V1ObjectMeta["deletionTimestamp"]
}>
export function WorkspaceStatus({ status, deletionTimestamp }: TWorkspaceStatusProps) {
  const displayStatus = determineDisplayStatus(status, deletionTimestamp)

  return <WorkspaceDisplayStatusBadge displayStatus={displayStatus} />
}

export function WorkspaceDisplayStatusBadge({
  displayStatus,
  compact,
}: {
  displayStatus: TWorkspaceDisplayStatus
  compact?: boolean
}) {
  const badgeProps = badgeOptionMappings[displayStatus]

  return <StatusBadge displayStatus={displayStatus} compact={compact} {...badgeProps} />
}

type TStatusBadgeProps = Readonly<{
  icon?: React.ReactNode
  color?: BoxProps["color"]
  displayStatus: TWorkspaceDisplayStatus
  compact?: boolean
}>
function StatusBadge({ icon, displayStatus, color, compact }: TStatusBadgeProps) {
  let s: string = displayStatus
  if (displayStatus === WorkspaceDisplayStatus.WaitingToInitialize) {
    s = "Waiting to Initialize"
  }

  return (
    <HStack
      w={compact ? "fit-content" : "full"}
      align="center"
      justify="start"
      gap="1"
      color={color}>
      {icon}
      {!compact && <Text fontWeight="medium">{s}</Text>}
    </HStack>
  )
}

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
