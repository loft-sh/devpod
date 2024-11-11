import { ProWorkspaceInstance } from "@/contexts"
import { CheckCircle, CircleDuotone, Clock, ExclamationTriangle, NotFound, Sleep } from "@/icons"
import { BoxProps, HStack, Text } from "@chakra-ui/react"
import React from "react"

export const InstancePhase = {
  Ready: "Ready",
  WaitingToInitialize: "",
  Sleeping: "Sleeping",
  Failed: "Failed",
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
}>
export function WorkspaceStatus({ status }: TWorkspaceStatusProps) {
  const displayStatus = determineDisplayStatus(status)
  const badgeProps = badgeOptionMappings[displayStatus]

  return <StatusBadge displayStatus={displayStatus} {...badgeProps} />
}

type TStatusBadgeProps = Readonly<{
  icon?: React.ReactNode
  color?: BoxProps["color"]
  displayStatus: TWorkspaceDisplayStatus
}>
function StatusBadge({ icon, displayStatus, color }: TStatusBadgeProps) {
  let s: string = displayStatus
  if (displayStatus === WorkspaceDisplayStatus.WaitingToInitialize) {
    s = "Waiting to Initialize"
  }

  return (
    <HStack w="full" align="center" justify="start" gap="1" color={color}>
      {icon}
      <Text fontWeight="medium">{s}</Text>
    </HStack>
  )
}

type TWorkspaceDisplayStatus = (typeof WorkspaceDisplayStatus)[keyof typeof WorkspaceDisplayStatus]

function determineDisplayStatus(status: ProWorkspaceInstance["status"]): TWorkspaceDisplayStatus {
  const phase = status?.phase
  const lastWorkspaceStatus = status?.lastWorkspaceStatus

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
