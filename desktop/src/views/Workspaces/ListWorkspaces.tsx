import { WORKSPACE_STATUSES } from "@/constants"
import { removeWorkspaceAction, stopWorkspaceAction } from "@/contexts/DevPodContext/workspaces"
import { Pause, Stack3D, Trash, WorkspaceStatus } from "@/icons"
import { ChevronDownIcon } from "@chakra-ui/icons"
import {
  Box,
  Button,
  Checkbox,
  FormLabel,
  HStack,
  Image,
  Menu,
  MenuButton,
  MenuDivider,
  MenuItemOption,
  MenuList,
  MenuOptionGroup,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Text,
  VStack,
  useDisclosure,
} from "@chakra-ui/react"
import { useId, useMemo, useState } from "react"
import { useNavigate } from "react-router"
import { useProviders, useWorkspaceStore, useWorkspaces } from "../../contexts"
import { exists } from "../../lib"
import { Routes } from "../../routes"
import { TProviderID, TWorkspace } from "../../types"
import { WorkspaceCard } from "./WorkspaceCard"
import { WorkspaceStatusBadge } from "./WorkspaceStatusBadge"

const SORT_OPTIONS = [
  "Recently Used",
  "Least Recently Used",
  "Recently Created",
  "Least Recently Created",
] as const
const DEFAULT_SORT_OPTION = "Recently Used"

type TWorkspacesInfo = Readonly<{
  workspaceCards: TWorkspace[]
}>

export function ListWorkspaces() {
  const { store } = useWorkspaceStore()
  const viewID = useId()
  const navigate = useNavigate()
  const [[providers]] = useProviders()
  const workspaces = useWorkspaces<TWorkspace>()
  const [selectedWorkspaces, setSelectedWorkspaces] = useState(new Set<string>())
  const [forceDelete, setForceDelete] = useState(false)
  const {
    isOpen: isDeleteOpen,
    onOpen: handleDeleteClicked,
    onClose: onDeleteClose,
  } = useDisclosure()

  const [providersFilter, setProvidersFilter] = useState<TProviderID[] | "all">("all")
  const [statusFilter, setStatusFilter] = useState<string[] | "all">("all")
  const [selectedSortOption, setSelectedSortOption] = useState(DEFAULT_SORT_OPTION)

  const { workspaceCards } = useMemo<TWorkspacesInfo>(() => {
    const empty: TWorkspacesInfo = { workspaceCards: [] }
    if (!exists(workspaces)) {
      return empty
    }

    const ret = workspaces.reduce<TWorkspacesInfo>((acc, workspace) => {
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

    ret.workspaceCards.sort((a, b) => {
      if (selectedSortOption === "Recently Used") {
        return new Date(a.lastUsed) > new Date(b.lastUsed) ? -1 : 1
      }

      if (selectedSortOption === "Least Recently Used") {
        return new Date(b.lastUsed) > new Date(a.lastUsed) ? -1 : 1
      }

      if (selectedSortOption === "Recently Created") {
        return new Date(a.creationTimestamp) > new Date(b.creationTimestamp) ? -1 : 1
      }

      if (selectedSortOption === "Least Recently Created") {
        return new Date(b.creationTimestamp) > new Date(a.creationTimestamp) ? -1 : 1
      }

      return 0
    })

    return ret
  }, [providersFilter, selectedSortOption, statusFilter, workspaces])

  const handleSelectionChanged = (workspaceID: string) => () => {
    setSelectedWorkspaces((curr) => {
      const updated = new Set(curr)
      if (updated.has(workspaceID)) {
        updated.delete(workspaceID)
      } else {
        updated.add(workspaceID)
      }

      return updated
    })
  }

  const handleSelectAllClicked = () => {
    setSelectedWorkspaces((curr) => {
      if (curr.size === workspaceCards.length) {
        return new Set()
      }

      return new Set(workspaceCards.map((workspace) => workspace.id))
    })
  }

  const handleStopAllClicked = () => {
    const allSelected = workspaces.filter((workspace) => selectedWorkspaces.has(workspace.id))
    for (const w of allSelected) {
      stopWorkspaceAction({
        workspaceID: w.id,
        streamID: viewID,
        store,
      })
    }

    setSelectedWorkspaces(new Set())
  }

  const handleDeleteAllClicked = () => {
    const allSelected = workspaces.filter((workspace) => selectedWorkspaces.has(workspace.id))
    for (const w of allSelected) {
      removeWorkspaceAction({
        workspaceID: w.id,
        streamID: viewID,
        force: forceDelete,
        store,
      })
    }
  }

  const allProvidersCount = Object.keys(providers ?? {}).length

  return (
    <>
      <VStack alignItems={"flex-start"} paddingBottom="6" w="full">
        <HStack justifyContent="space-between" w="full">
          <HStack>
            <Checkbox
              id="select-all"
              isIndeterminate={
                selectedWorkspaces.size > 0 && selectedWorkspaces.size < workspaceCards.length
              }
              isChecked={
                workspaceCards.length > 0 && selectedWorkspaces.size === workspaceCards.length
              }
              onChange={handleSelectAllClicked}
            />
            <FormLabel whiteSpace="nowrap" paddingTop="2" htmlFor="select-all" color="gray.500">
              {selectedWorkspaces.size === 0
                ? "Select all"
                : ` ${selectedWorkspaces.size} of ${workspaceCards.length} selected`}
            </FormLabel>
            {selectedWorkspaces.size > 0 && (
              <>
                <Button
                  variant="ghost"
                  leftIcon={<Pause boxSize={4} />}
                  onClick={handleStopAllClicked}>
                  Stop
                </Button>
                <Button
                  variant="ghost"
                  colorScheme="red"
                  leftIcon={<Trash boxSize={4} />}
                  onClick={handleDeleteClicked}>
                  Delete
                </Button>
              </>
            )}
          </HStack>

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
                  {Object.entries(providers ?? {}).map(([providerID, provider]) => (
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

            <Menu closeOnSelect={false} offset={[0, 2]}>
              <MenuButton
                as={Button}
                variant="outline"
                leftIcon={<WorkspaceStatus boxSize={4} color="gray.600" />}>
                Status ({getCurrentFilterCount(statusFilter, WORKSPACE_STATUSES.length)}/
                {WORKSPACE_STATUSES.length})
              </MenuButton>
              <MenuList>
                <MenuItemOption
                  isChecked={
                    statusFilter.includes("all") ||
                    statusFilter.length === WORKSPACE_STATUSES.length
                  }
                  onClick={() => setStatusFilter((curr) => (curr === "all" ? [] : "all"))}
                  key="all"
                  value="all">
                  Select All
                </MenuItemOption>
                <MenuOptionGroup
                  value={
                    statusFilter === "all"
                      ? (WORKSPACE_STATUSES as unknown as string[])
                      : statusFilter
                  }
                  onChange={(value) => setStatusFilter(typeof value === "string" ? [value] : value)}
                  type="checkbox">
                  <MenuDivider />
                  {WORKSPACE_STATUSES.map((status) => (
                    <MenuItemOption key={status} value={status}>
                      <HStack>
                        <WorkspaceStatusBadge
                          status={status}
                          isLoading={false}
                          hasError={false}
                          showText={false}
                        />{" "}
                        <Text> {status}</Text>
                      </HStack>
                    </MenuItemOption>
                  ))}
                </MenuOptionGroup>
              </MenuList>
            </Menu>

            <Menu offset={[0, 2]}>
              <MenuButton as={Button} variant="outline" rightIcon={<ChevronDownIcon boxSize={4} />}>
                Sort by: {selectedSortOption}
              </MenuButton>
              <MenuList>
                <MenuOptionGroup
                  type="radio"
                  value={selectedSortOption}
                  onChange={(value) =>
                    setSelectedSortOption(
                      (Array.isArray(value) ? value[0] : value) ?? DEFAULT_SORT_OPTION
                    )
                  }>
                  {SORT_OPTIONS.map((option) => (
                    <MenuItemOption key={option} value={option}>
                      {option}
                    </MenuItemOption>
                  ))}
                </MenuOptionGroup>
              </MenuList>
            </Menu>
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
                isSelected={selectedWorkspaces.has(workspace.id)}
                onSelectionChange={handleSelectionChanged(workspace.id)}
              />
            ))
          )}
        </Box>
      </VStack>

      <Modal onClose={onDeleteClose} isOpen={isDeleteOpen} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Delete {selectedWorkspaces.size} Workspaces</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            Deleting the workspaces will erase all state. Are you sure you want to delete the
            selected workspaces?
            <Box marginTop={"2.5"}>
              <Checkbox checked={forceDelete} onChange={(e) => setForceDelete(e.target.checked)}>
                Force Delete
              </Checkbox>
            </Box>
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onDeleteClose}>Close</Button>
              <Button
                colorScheme="red"
                onClick={() => {
                  handleDeleteAllClicked()
                  onDeleteClose()
                  setSelectedWorkspaces(new Set())
                }}>
                Delete
              </Button>
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  )
}

function getCurrentFilterCount(filter: string[] | "all", total: number) {
  if (filter === "all") {
    return total
  }

  return filter.length
}
