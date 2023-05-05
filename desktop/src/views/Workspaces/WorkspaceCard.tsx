import {
  Box,
  Button,
  Card,
  CardFooter,
  CardHeader,
  Checkbox,
  Heading,
  HStack,
  Icon,
  IconButton,
  Image,
  InputGroup,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Portal,
  Select,
  Stack,
  Text,
  Tooltip,
  useDisclosure,
  VStack,
} from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import dayjs from "dayjs"
import { useCallback, useMemo, useState } from "react"
import { HiClock, HiOutlineCode } from "react-icons/hi"
import { useNavigate } from "react-router"
import { client } from "../../client"
import { IconTag } from "../../components"
import { TActionID, useWorkspace, useWorkspaceActions } from "../../contexts"
import { Ellipsis, Pause, Play, Stack3D, Trash, ArrowPath } from "../../icons"
import { CodeJPG } from "../../images"
import { getIDEDisplayName } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import { Routes } from "../../routes"
import { TWorkspace, TWorkspaceID } from "../../types"
import { getSourceName, getIDEName } from "./helpers"

type TWorkspaceCardProps = Readonly<{
  workspaceID: TWorkspaceID
  onSelectionChange?: (isSelected: boolean) => void
}>

export function WorkspaceCard({ workspaceID, onSelectionChange }: TWorkspaceCardProps) {
  const [forceDelete, setForceDelete] = useState<boolean>(false)
  const navigate = useNavigate()
  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()
  const { isOpen: isRebuildOpen, onOpen: onRebuildOpen, onClose: onRebuildClose } = useDisclosure()
  const workspace = useWorkspace(workspaceID)
  const [ideName, setIdeName] = useState<string | undefined>(workspace.data?.ide?.name ?? undefined)
  const [rebuildMenuItemPointerEvents, setRebuildMenuItemPointerEvents] = useState<"auto" | "none">(
    "auto"
  )

  const navigateToAction = useCallback(
    (actionID: TActionID | undefined) => {
      if (actionID !== undefined && actionID !== "") {
        navigate(Routes.toAction(actionID))
      }
    },
    [navigate]
  )

  const handleOpenWithIDEClicked = useCallback(
    (id: TWorkspaceID) => async () => {
      if (!ideName) {
        return
      }

      const actionID = workspace.start({ id, ideConfig: { name: ideName } })
      navigateToAction(actionID)
    },
    [ideName, workspace, navigateToAction]
  )

  const isLoading = useMemo(() => {
    if (workspace.current?.status === "pending") {
      return true
    }

    return false
  }, [workspace])

  const isOpenDisabled = workspace.data?.status === "Busy"
  const isOpenDisabledReason =
    "Cannnot open this workspace because it is busy. If this doesn't change, try to force delete and recreate it."

  if (workspace.data === undefined) {
    return null
  }

  const { id, picture, ide } = workspace.data

  return (
    <>
      <Card
        key={id}
        direction="row"
        width="full"
        maxWidth="60rem"
        overflow="hidden"
        variant="outline"
        maxHeight="48">
        <Image
          objectFit="cover"
          maxHeight={"full"}
          width={"300px"}
          maxWidth={"300px"}
          style={{ aspectRatio: "2 / 1" }}
          src={picture ?? CodeJPG}
          fallbackSrc={CodeJPG}
          alt="Project Image"
        />
        <Stack width="full" justifyContent={"space-between"}>
          <WorkspaceCardHeader
            workspace={workspace.data}
            isLoading={isLoading}
            onSelectionChange={onSelectionChange}
            onActionIndicatorClicked={navigateToAction}
          />

          <CardFooter padding="none" paddingBottom={4}>
            <HStack spacing="2" width="full" justifyContent="end" paddingRight={"10px"}>
              <Tooltip label={isOpenDisabled ? isOpenDisabledReason : undefined}>
                <Button
                  aria-label="Start workspace"
                  variant="primary"
                  leftIcon={<Icon as={HiOutlineCode} boxSize={5} />}
                  isDisabled={isOpenDisabled}
                  onClick={() => {
                    const actionID = workspace.start({ id, ideConfig: ide })
                    navigateToAction(actionID)
                  }}
                  isLoading={isLoading}>
                  Open
                </Button>
              </Tooltip>
              <Menu placement="top">
                <MenuButton
                  as={IconButton}
                  aria-label="More actions"
                  variant="ghost"
                  colorScheme="gray"
                  isDisabled={isLoading}
                  icon={<Ellipsis transform={"rotate(90deg)"} boxSize={5} />}
                />
                <Portal>
                  <MenuList minWidth="72">
                    <InputGroup
                      paddingRight={3}
                      _hover={{ backgroundColor: "gray.100", cursor: "pointer" }}>
                      <Button
                        variant="ghost"
                        transition={"none"}
                        borderRadius={0}
                        fontWeight={"normal"}
                        leftIcon={<Play boxSize={4} />}
                        onClick={handleOpenWithIDEClicked(workspace.data.id)}>
                        Start with
                      </Button>
                      <Select
                        size="sm"
                        maxWidth={40}
                        overflow="hidden"
                        textOverflow="ellipsis"
                        borderRadius={0}
                        whiteSpace="nowrap"
                        textTransform="capitalize"
                        onChange={(e) => setIdeName(e.target.value)}
                        onFocus={() => setRebuildMenuItemPointerEvents("none")}
                        onBlur={() => setRebuildMenuItemPointerEvents("auto")}
                        value={ideName}>
                        {idesQuery.data?.map((ide) => (
                          <option key={ide.name} value={ide.name!}>
                            {getIDEDisplayName(ide)}
                          </option>
                        ))}
                      </Select>
                    </InputGroup>
                    <MenuItem
                      style={{ pointerEvents: rebuildMenuItemPointerEvents }}
                      icon={<ArrowPath boxSize={4} />}
                      onClick={onRebuildOpen}>
                      Rebuild
                    </MenuItem>
                    <MenuItem
                      onClick={() => workspace.stop()}
                      icon={<Pause boxSize={4} />}
                      isDisabled={workspace.data.status !== "Running"}>
                      Stop
                    </MenuItem>
                    <MenuItem
                      fontWeight="normal"
                      icon={<Trash boxSize={4} />}
                      onClick={() => onDeleteOpen()}>
                      Delete
                    </MenuItem>
                  </MenuList>
                </Portal>
              </Menu>
            </HStack>
          </CardFooter>
        </Stack>
      </Card>

      <Modal onClose={onRebuildClose} isOpen={isRebuildOpen} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Rebuild Workspace</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            Rebuilding the workspace will erase all state saved in the docker container overlay.
            This means you might need to reinstall or reconfigure certain applications. State in
            docker volumes is persisted. Are you sure you want to rebuild {workspace.data.id}?
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onRebuildClose}>Close</Button>
              <Button
                colorScheme={"primary"}
                onClick={async () => {
                  const actionID = workspace.rebuild()
                  onRebuildClose()
                  navigateToAction(actionID)
                }}>
                Rebuild
              </Button>
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <Modal onClose={onDeleteClose} isOpen={isDeleteOpen} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Delete Workspace</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            Deleting the workspace will erase all state. Are you sure you want to delete{" "}
            {workspace.data.id}?
            <Box marginTop={"10px"}>
              <Checkbox checked={forceDelete} onChange={(e) => setForceDelete(e.target.checked)}>
                Force Delete the Workspace
              </Checkbox>
            </Box>
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onDeleteClose}>Close</Button>
              <Button
                colorScheme={"red"}
                onClick={async () => {
                  workspace.remove(forceDelete)
                  onDeleteClose()
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

type TWorkspaceCardHeaderProps = Readonly<{
  workspace: TWorkspace
  isLoading: boolean
  onActionIndicatorClicked: (actionID: TActionID | undefined) => void
  onSelectionChange?: (isSelected: boolean) => void
}>
function WorkspaceCardHeader({
  workspace,
  isLoading,
  onSelectionChange,
  onActionIndicatorClicked,
}: TWorkspaceCardHeaderProps) {
  const { id, status, provider, ide, lastUsed, source } = workspace
  const workspaceActions = useWorkspaceActions(id)

  const errorActionID = useMemo(() => {
    if (!workspaceActions?.length || workspaceActions[0]?.status !== "error") {
      return undefined
    }

    return workspaceActions[0]?.id
  }, [workspaceActions])

  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })

  return (
    <CardHeader display="flex" flexDirection="column">
      <VStack align="start" spacing={0}>
        <HStack justifyContent="space-between">
          <Heading size="md">
            <HStack alignItems="center">
              <Text fontWeight="bold">{id}</Text>
              <Tooltip
                label={
                  errorActionID
                    ? "Workspace encountered an error"
                    : isLoading
                    ? `Workspace is loading`
                    : `Workspace is ${status ?? "Pending"}`
                }>
                <Box
                  as={"span"}
                  onClick={() => {
                    if (errorActionID) {
                      onActionIndicatorClicked(errorActionID)
                    } else if (isLoading) {
                      onActionIndicatorClicked(id)
                    }
                  }}
                  cursor={errorActionID || isLoading ? "pointer" : undefined}
                  backgroundColor={
                    errorActionID
                      ? "red"
                      : isLoading
                      ? "orange"
                      : status === "Running"
                      ? "green"
                      : "orange"
                  }
                  borderRadius={"full"}
                  width={"10px"}
                  height={"10px"}
                />
              </Tooltip>
            </HStack>
          </Heading>
          {onSelectionChange !== undefined && (
            <Checkbox onChange={(e) => onSelectionChange(e.target.checked)} />
          )}
        </HStack>
        {source !== null && (
          <Text fontSize="sm" color="gray.500" userSelect="auto">
            {getSourceName(source)}
          </Text>
        )}
      </VStack>

      <HStack rowGap={2} marginTop={4} flexWrap="wrap" alignItems="center">
        <IconTag
          icon={<Stack3D />}
          label={provider?.name ?? "No provider"}
          infoText={provider?.name ? `Uses provider ${provider.name}` : undefined}
        />
        <IconTag
          icon={<Icon as={HiOutlineCode} />}
          label={getIDEName(ide, idesQuery.data)}
          infoText={`Will be opened in ${getIDEName(ide, idesQuery.data)}`}
        />
        <IconTag
          icon={<Icon as={HiClock} />}
          label={dayjs(new Date(lastUsed)).fromNow()}
          infoText={`Last used ${dayjs(new Date(lastUsed)).fromNow()}`}
        />
      </HStack>
    </CardHeader>
  )
}
