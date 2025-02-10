import { removeWorkspaceAction, stopWorkspaceAction } from "@/contexts/DevPodContext/workspaces"
import { Stack3D } from "@/icons"
import {
  Box,
  Button,
  HStack,
  Image,
  Menu,
  MenuButton,
  MenuDivider,
  MenuItemOption,
  MenuList,
  MenuOptionGroup,
  Text,
  useDisclosure,
  VStack,
} from "@chakra-ui/react"
import { useCallback, useEffect, useId, useMemo, useState } from "react"
import { useNavigate } from "react-router"
import { useProviders, useWorkspaces, useWorkspaceStore } from "@/contexts"
import { exists, useSelection, useSortWorkspaces } from "@/lib"
import { Routes } from "@/routes"
import { TProviderID, TWorkspace } from "@/types"
import { WorkspaceCard } from "./WorkspaceCard"
import { WorkspaceListSelection } from "@/components/ListSelection"
import { DeleteWorkspacesModal } from "@/components/DeleteWorkspacesModal"
import { DEFAULT_SORT_WORKSPACE_MODE } from "@/lib/useSortWorkspaces"
import { TWorkspaceStatusFilterState, WorkspaceSorter, WorkspaceStatusFilter } from "@/components"

type TWorkspacesInfo = Readonly<{
  workspaceCards: TWorkspace[]
}>

export function ListWorkspaces() {
  const { store } = useWorkspaceStore()
  const viewID = useId()
  const navigate = useNavigate()
  const [[providers]] = useProviders()
  const workspaces = useWorkspaces<TWorkspace>()

  const selection = useSelection()

  const {
    isOpen: isDeleteOpen,
    onOpen: handleDeleteClicked,
    onClose: onDeleteClose,
  } = useDisclosure()

  const [providersFilter, setProvidersFilter] = useState<TProviderID[] | "all">("all")
  const [statusFilter, setStatusFilter] = useState<TWorkspaceStatusFilterState>("all")

  const [selectedSortOption, setSelectedSortOption] = useState(DEFAULT_SORT_WORKSPACE_MODE)
  const sortedWorkspaces = useSortWorkspaces(workspaces, selectedSortOption)

  const { workspaceCards } = useMemo<TWorkspacesInfo>(() => {
    const empty: TWorkspacesInfo = { workspaceCards: [] }
    if (!exists(sortedWorkspaces)) {
      return empty
    }

    return sortedWorkspaces.reduce<TWorkspacesInfo>((acc, workspace) => {
      const { id } = workspace
      if (!exists(id)) {
        return acc
      }

      if (
        workspace.provider?.name &&
        providersFilter !== "all" &&
        !providersFilter.includes(workspace.provider.name)
      ) {
        return acc
      }

      if (statusFilter !== "all" && !statusFilter.includes(workspace.status as string)) {
        return acc
      }

      acc.workspaceCards.push(workspace)

      return acc
    }, empty)
  }, [providersFilter, statusFilter, sortedWorkspaces])

  const workspaceIds = useMemo(() => {
    return workspaceCards.map((w) => w.id)
  }, [workspaceCards])

  useEffect(() => {
    selection.prune(workspaceIds)
  }, [selection, workspaceIds])

  const handleSelectionChanged = useCallback(
    (workspaceID: string) => () => selection.toggleSelection(workspaceID),
    [selection]
  )

  const handleSelectAllClicked = useCallback(() => {
    selection.toggleSelectAll(workspaceIds)
  }, [workspaceIds, selection])

  const handleStopAllClicked = useCallback(() => {
    const allSelected = workspaces.filter((workspace) => selection.has(workspace.id))
    for (const w of allSelected) {
      stopWorkspaceAction({
        workspaceID: w.id,
        streamID: viewID,
        store,
      })
    }

    selection.clear()
  }, [selection, store, viewID, workspaces])

  const handleDeleteAllClicked = useCallback(
    (forceDelete: boolean) => {
      const allSelected = workspaces.filter((workspace) => selection.has(workspace.id))
      for (const w of allSelected) {
        removeWorkspaceAction({
          workspaceID: w.id,
          streamID: viewID,
          force: forceDelete,
          store,
        })
      }
      selection.clear()
    },
    [selection, workspaces, viewID, store]
  )

  const availableProviders = Object.entries(providers ?? {}).filter(
    ([, provider]) => !provider.isProxyProvider
  )
  const allProvidersCount = availableProviders.length

  return (
    <>
      <VStack alignItems={"flex-start"} paddingBottom="6" w="full">
        <HStack justifyContent="space-between" w="full">
          <WorkspaceListSelection
            totalAmount={workspaceCards.length}
            selectionAmount={selection.size}
            handleSelectAllClicked={handleSelectAllClicked}
            handleStopAllClicked={handleStopAllClicked}
            handleDeleteClicked={handleDeleteClicked}
          />

          <HStack>
            <Menu closeOnSelect={false} offset={[0, 2]}>
              <MenuButton as={Button} variant="outline" leftIcon={<Stack3D boxSize={4} />}>
                Providers ({getCurrentFilterCount(providersFilter, allProvidersCount)}/
                {allProvidersCount})
              </MenuButton>
              <MenuList>
                <MenuItemOption
                  isChecked={
                    providersFilter.includes("all") || providersFilter.length === allProvidersCount
                  }
                  onClick={() => setProvidersFilter((curr) => (curr === "all" ? [] : "all"))}
                  value="all">
                  Select All
                </MenuItemOption>
                <MenuOptionGroup
                  value={providersFilter === "all" ? Object.keys(providers ?? {}) : providersFilter}
                  onChange={(value) =>
                    setProvidersFilter(typeof value === "string" ? [value] : value)
                  }
                  type="checkbox">
                  <MenuDivider />
                  {availableProviders.map(([providerID, provider]) => (
                    <MenuItemOption key={providerID} value={providerID}>
                      <HStack>
                        {provider.config?.icon ? (
                          <Image src={provider.config.icon} boxSize={4} />
                        ) : (
                          <Box boxSize={4} />
                        )}
                        <Text>{providerID}</Text>
                      </HStack>
                    </MenuItemOption>
                  ))}
                </MenuOptionGroup>
              </MenuList>
            </Menu>

            <WorkspaceStatusFilter statusFilter={statusFilter} setStatusFilter={setStatusFilter} />

            <WorkspaceSorter sortMode={selectedSortOption} setSortMode={setSelectedSortOption} />
          </HStack>
        </HStack>

        <Box w="full" h="full">
          {workspaceCards.length === 0 ? (
            <VStack paddingTop="6">
              <Text>No workspaces for selection found. Click here to create one</Text>
              <Button onClick={() => navigate(Routes.WORKSPACE_CREATE)}>Create Workspace</Button>
            </VStack>
          ) : (
            workspaceCards.map((workspace) => (
              <WorkspaceCard
                key={workspace.id}
                workspaceID={workspace.id}
                isSelected={selection.has(workspace.id)}
                onSelectionChange={handleSelectionChanged(workspace.id)}
              />
            ))
          )}
        </Box>
      </VStack>

      <DeleteWorkspacesModal
        isOpen={isDeleteOpen}
        onCloseRequested={onDeleteClose}
        onDeleteRequested={handleDeleteAllClicked}
        amount={selection.size}
      />
    </>
  )
}

function getCurrentFilterCount(filter: string[] | "all", total: number) {
  if (filter === "all") {
    return total
  }

  return filter.length
}
