import { TWorkspaceStatusFilterState, WorkspaceSorter, WorkspaceStatusFilter } from "@/components"
import { DeleteWorkspacesModal } from "@/components/DeleteWorkspacesModal"
import { WorkspaceListSelection } from "@/components/ListSelection"
import {
  ProWorkspaceInstance,
  useProContext,
  useTemplates,
  useWorkspaces,
  useWorkspaceStore,
} from "@/contexts"
import { removeWorkspaceAction, stopWorkspaceAction } from "@/contexts/DevPodContext/workspaces"
import { IWorkspaceStore } from "@/contexts/DevPodContext/workspaceStore"
import { DevPodIcon } from "@/icons"
import emptyWorkspacesImage from "@/images/empty_workspaces.svg"
import {
  DEFAULT_SORT_WORKSPACE_MODE,
  useSelection,
  useSortProWorkspaces,
  useStopWorkspaceModal,
} from "@/lib"
import { Routes } from "@/routes"
import { determineDisplayStatus } from "@/views/Pro/Workspace/WorkspaceStatus"
import {
  Button,
  Center,
  Container,
  Heading,
  HStack,
  Image,
  List,
  ListItem,
  Spinner,
  Text,
  useDisclosure,
  VStack,
} from "@chakra-ui/react"
import { useCallback, useEffect, useId, useMemo, useState } from "react"
import { useNavigate } from "react-router"
import { WorkspaceInstanceCard } from "./Workspace"

import { WorkspaceOwnerFilter } from "@/components/WorkspaceOwnerFilter"
import EmptyImage from "@/images/empty-default.svg"
import { ManagementV1Self } from "@loft-enterprise/client/gen/models/managementV1Self"

export function ListWorkspaces() {
  const { store } = useWorkspaceStore<IWorkspaceStore<string, ProWorkspaceInstance>>()
  const instances = useWorkspaces<ProWorkspaceInstance>()
  const viewID = useId()
  const { host, isLoadingWorkspaces, managementSelfQuery, ownerFilter, setOwnerFilter } =
    useProContext()
  const navigate = useNavigate()
  const { data: templates } = useTemplates()

  const [statusFilter, setStatusFilter] = useState<TWorkspaceStatusFilterState>("all")

  const filteredWorkspaces = useMemo(() => {
    let retInstances = instances

    // owner filter
    if (ownerFilter == "self") {
      retInstances = retInstances.filter((i) => isOwner(i, managementSelfQuery.data))
    }

    // status filter
    if (statusFilter != "all") {
      retInstances = retInstances.filter((i) =>
        statusFilter.includes(determineDisplayStatus(i.status, i.metadata?.deletionTimestamp))
      )
    }

    return retInstances
  }, [instances, managementSelfQuery.data, ownerFilter, statusFilter])

  const [selectedSortOption, setSelectedSortOption] = useState(DEFAULT_SORT_WORKSPACE_MODE)
  const sortedWorkspaces = useSortProWorkspaces(filteredWorkspaces, selectedSortOption)

  const selection = useSelection()

  const { isOpen: isDeleteOpen, onOpen: openDeleteModal, onClose: onDeleteClose } = useDisclosure()

  const instanceIDs = useMemo(() => {
    return (sortedWorkspaces ?? []).map((i) => i.id)
  }, [sortedWorkspaces])

  useEffect(() => {
    selection.prune(instanceIDs)
  }, [instanceIDs, selection])

  const handleDeleteAllClicked = useCallback(
    (forceDelete: boolean) => {
      const allSelected = instances.filter((workspace) => selection.has(workspace.id))
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
    [selection, instances, viewID, store]
  )

  const stopAll = useCallback(() => {
    const allSelected = instances.filter((workspace) => selection.has(workspace.id))
    for (const w of allSelected) {
      stopWorkspaceAction({
        workspaceID: w.id,
        streamID: viewID,
        store,
      })
    }

    selection.clear()
  }, [instances, selection, store, viewID])

  const handleSelectAllClicked = useCallback(() => {
    selection.toggleSelectAll(instanceIDs)
  }, [instanceIDs, selection])

  const handleCreateClicked = useCallback(() => {
    if (templates?.presets.length) {
      navigate(Routes.toProSelectPreset(host))
    } else {
      navigate(Routes.toProWorkspaceCreate(host))
    }
  }, [navigate, templates, host])

  const { modal: stopModal, open: openStopModal } = useStopWorkspaceModal(
    useCallback(
      (close) => {
        stopAll()
        close()
      },
      [stopAll]
    )
  )

  const handleStopAllClicked = useCallback(() => {
    const nonRunningWorkspace = instances.find(
      (i) => selection.has(i.id) && i.status?.lastWorkspaceStatus !== "Running"
    )

    if (nonRunningWorkspace) {
      openStopModal()
    } else {
      stopAll()
    }
  }, [stopAll, openStopModal, instances, selection])

  const hasWorkspaces = instances.length > 0

  return (
    <>
      <VStack align="start" gap="4" w="full" h="full">
        {hasWorkspaces ? (
          <>
            <HStack align="center" justify="space-between" mb="8" w="full">
              <Heading fontWeight="thin">Workspaces</Heading>
              <Button
                variant="outline"
                colorScheme="primary"
                leftIcon={<DevPodIcon boxSize={5} />}
                onClick={handleCreateClicked}>
                Create Workspace
              </Button>
            </HStack>
            <HStack align={"center"} justify={"space-between"} w={"full"}>
              <WorkspaceListSelection
                totalAmount={filteredWorkspaces.length}
                selectionAmount={selection.size}
                handleSelectAllClicked={handleSelectAllClicked}
                handleStopAllClicked={handleStopAllClicked}
                handleDeleteClicked={openDeleteModal}
              />
              <HStack align={"center"}>
                <WorkspaceOwnerFilter ownerFilter={ownerFilter} setOwnerFilter={setOwnerFilter} />
                <WorkspaceStatusFilter
                  variant={"pro"}
                  statusFilter={statusFilter}
                  setStatusFilter={setStatusFilter}
                />
                <WorkspaceSorter
                  sortMode={selectedSortOption}
                  setSortMode={setSelectedSortOption}
                />
              </HStack>
            </HStack>
            <List w="full" h={"full"} mb="4">
              {!sortedWorkspaces?.length && (
                <VStack
                  w={"full"}
                  h={"full"}
                  justifyContent={"center"}
                  alignItems={"center"}
                  flexGrow={1}>
                  <Image src={EmptyImage} />
                  <Text fontWeight={"semibold"} fontSize={"sm"} color={"text.secondary"}>
                    No items found
                  </Text>
                </VStack>
              )}
              {sortedWorkspaces?.map((instance) => (
                <ListItem key={instance.id}>
                  <WorkspaceInstanceCard
                    host={host}
                    isSelected={selection.has(instance.id)}
                    onSelectionChange={(isSelected) =>
                      selection.setSelected(instance.id, isSelected)
                    }
                    instanceName={instance.id}
                  />
                </ListItem>
              ))}
            </List>
          </>
        ) : isLoadingWorkspaces ? (
          <Center w="full" h="60%" flexFlow="column nowrap">
            <Spinner size="xl" thickness="4px" speed="1s" color="gray.600" />
            <Text mt="4">Loading Workspaces...</Text>
          </Center>
        ) : (
          <Container maxW="container.lg" h="full">
            <VStack align="center" justify="center" w="full" h="full">
              <Heading fontWeight="thin" color="gray.600">
                Create a DevPod Workspace
              </Heading>
              <Image src={emptyWorkspacesImage} w="100%" h="40vh" my="12" />

              <Button
                variant="solid"
                colorScheme="primary"
                leftIcon={<DevPodIcon boxSize={5} />}
                onClick={handleCreateClicked}>
                Create Workspace
              </Button>
            </VStack>
          </Container>
        )}
      </VStack>

      <DeleteWorkspacesModal
        isOpen={isDeleteOpen}
        onCloseRequested={onDeleteClose}
        onDeleteRequested={handleDeleteAllClicked}
        amount={selection.size}
      />

      {stopModal}
    </>
  )
}

function isOwner(instance: ProWorkspaceInstance, self: ManagementV1Self | undefined): boolean {
  if (!self) {
    return false
  }

  if (!instance.spec?.owner) {
    return false
  }

  const owner = instance.spec.owner
  if (self.status?.user?.name == owner.user) {
    return true
  }
  if (owner.team && self.status?.user?.teams?.find((team) => team.name == owner.team)) {
    return true
  }

  return false
}
