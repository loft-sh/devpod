import {
  Button,
  HStack,
  Menu,
  MenuButton,
  MenuDivider,
  MenuItemOption,
  MenuList,
  MenuOptionGroup,
  Text,
} from "@chakra-ui/react"
import { WorkspaceStatus } from "@/icons"
import { WORKSPACE_STATUSES } from "@/constants"
import { WorkspaceStatusBadge } from "@/views/Workspaces/WorkspaceStatusBadge"
import { useCallback } from "react"
import { WorkspaceDisplayStatusBadge } from "@/views/Pro/Workspace/WorkspaceStatus"
import { TWorkspace } from "@/types"
import { TWorkspaceDisplayStatus, WorkspaceDisplayStatus } from "@/views/Pro/Workspace/status"

export type TWorkspaceStatusFilterState = string[] | "all"

export function WorkspaceStatusFilter({
  statusFilter,
  setStatusFilter,
  variant = "oss",
}: {
  statusFilter: TWorkspaceStatusFilterState
  setStatusFilter: (statusFilter: TWorkspaceStatusFilterState) => void
  variant?: "oss" | "pro"
}) {
  const availableStatuses =
    variant === "oss" ? WORKSPACE_STATUSES : Object.values(WorkspaceDisplayStatus)

  const onSelectAll = useCallback(() => {
    if (statusFilter === "all") {
      setStatusFilter([])
    } else {
      setStatusFilter("all")
    }
  }, [statusFilter, setStatusFilter])

  const onChange = useCallback(
    (value: string | string[]) => {
      setStatusFilter(typeof value === "string" ? [value] : value)
    },
    [setStatusFilter]
  )

  return (
    <Menu closeOnSelect={false} offset={[0, 2]}>
      <MenuButton
        as={Button}
        variant="outline"
        leftIcon={<WorkspaceStatus boxSize={4} color="gray.600" />}>
        Status ({getCurrentFilterCount(statusFilter, availableStatuses.length)}/
        {availableStatuses.length})
      </MenuButton>
      <MenuList>
        <MenuItemOption
          isChecked={
            statusFilter.includes("all") || statusFilter.length === availableStatuses.length
          }
          onClick={onSelectAll}
          key="all"
          value="all">
          Select All
        </MenuItemOption>
        <MenuOptionGroup
          value={statusFilter === "all" ? (availableStatuses as unknown as string[]) : statusFilter}
          onChange={onChange}
          type="checkbox">
          <MenuDivider />
          {availableStatuses.map((status) => (
            <MenuItemOption key={status} value={status}>
              <HStack>
                {variant === "oss" ? (
                  <WorkspaceStatusBadge
                    status={status as TWorkspace["status"]}
                    isLoading={false}
                    hasError={false}
                    showText={false}
                  />
                ) : (
                  <WorkspaceDisplayStatusBadge
                    compact={true}
                    displayStatus={status as TWorkspaceDisplayStatus}
                  />
                )}{" "}
                <Text> {status || "Waiting to Initialize"}</Text>
              </HStack>
            </MenuItemOption>
          ))}
        </MenuOptionGroup>
      </MenuList>
    </Menu>
  )
}

function getCurrentFilterCount(filter: TWorkspaceStatusFilterState, total: number) {
  if (filter === "all") {
    return total
  }

  return filter.length
}
