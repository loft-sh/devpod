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
  Tag,
  TagLabel,
  Text,
  Tooltip,
  useColorModeValue,
  useDisclosure,
} from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import dayjs from "dayjs"
import { useCallback, useState } from "react"
import { HiClock, HiOutlineCode } from "react-icons/hi"
import { useNavigate } from "react-router"
import { client } from "../../client"
import { TActionID, useWorkspace } from "../../contexts"
import { Ellipsis, Pause, Play, Stack3D, Trash, ArrowPath } from "../../icons"
import { CodeJPG } from "../../images"
import { QueryKeys } from "../../queryKeys"
import { Routes } from "../../routes"
import { TIDEs, TWorkspace, TWorkspaceID } from "../../types"

type TWorkspaceCardProps = Readonly<{
  workspaceID: TWorkspaceID
  onSelectionChange?: (isSelected: boolean) => void
}>

export function WorkspaceCard({ workspaceID, onSelectionChange }: TWorkspaceCardProps) {
  const navigate = useNavigate()
  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()
  const { isOpen: isRebuildOpen, onOpen: onRebuildOpen, onClose: onRebuildClose } = useDisclosure()
  const workspace = useWorkspace(workspaceID)
  const [ideName, setIdeName] = useState<string | undefined>(workspace.data?.ide?.name ?? undefined)
  const tagColor = useColorModeValue("gray.700", "gray.300")

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

  if (workspace.data === undefined) {
    return null
  }

  const { id, provider, picture, status, ide } = workspace.data

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
          style={{ aspectRatio: "2 / 1" }}
          src={picture ?? CodeJPG}
          alt="Project Image"
        />

        <Stack width="full" justifyContent={"space-between"}>
          <CardHeader display="flex" flexDirection="column">
            <HStack justifyContent="space-between">
              <Heading size="md">
                <HStack alignItems="center">
                  <Text fontWeight="bold">{id}</Text>
                  <Tooltip label={`Workspace is ${status ?? "Pending"}`}>
                    <Box
                      as={"span"}
                      backgroundColor={status === "Running" ? "green" : "orange"}
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
            <HStack rowGap={2} marginTop={4} flexWrap="wrap" alignItems="center">
              <Tag borderRadius="full" color={tagColor}>
                <Stack3D boxSize={4} />
                <TagLabel marginLeft={2}>{provider?.name ?? "No provider"}</TagLabel>
              </Tag>
              <Tag borderRadius="full" color={tagColor}>
                <Icon boxSize={4} as={HiOutlineCode} />
                <TagLabel marginLeft={2}>{getIDEName(workspace.data.ide, idesQuery.data)}</TagLabel>
              </Tag>
              <Tag marginRight={2} borderRadius="full" color={tagColor}>
                <Icon as={HiClock} />
                <TagLabel marginLeft={2}>
                  {dayjs(new Date(workspace.data.lastUsed)).fromNow()}
                </TagLabel>
              </Tag>
            </HStack>
          </CardHeader>

          <CardFooter padding="none" paddingBottom={4}>
            <HStack spacing="2" width="full" justifyContent="end" paddingRight={"10px"}>
              <Button
                aria-label="Start workspace"
                variant="primary"
                leftIcon={<Icon as={HiOutlineCode} boxSize={5} />}
                isDisabled={
                  workspace.data.status === "Busy" || workspace.data.status === "NotFound"
                }
                onClick={() => {
                  const actionID = workspace.start({ id, ideConfig: ide })
                  navigateToAction(actionID)
                }}
                isLoading={
                  workspace.current?.name === "start" && workspace.current.status === "pending"
                }>
                Open
              </Button>
              <Menu placement="top">
                <Tooltip label="More Actions">
                  <MenuButton
                    as={IconButton}
                    aria-label="More actions"
                    variant="ghost"
                    colorScheme="gray"
                    icon={<Ellipsis transform={"rotate(90deg)"} boxSize={5} />}
                  />
                </Tooltip>
                <Portal>
                  <MenuList>
                    <InputGroup paddingRight={3}>
                      <Button
                        colorScheme={"primary"}
                        variant="ghost"
                        fontWeight={"normal"}
                        color="primary"
                        leftIcon={<Play boxSize={4} />}
                        onClick={handleOpenWithIDEClicked(workspace.data.id)}>
                        Start with
                      </Button>
                      <Select
                        maxWidth={40}
                        overflow="hidden"
                        textOverflow="ellipsis"
                        whiteSpace="nowrap"
                        onChange={(e) => setIdeName(e.target.value)}
                        value={ideName}>
                        {idesQuery.data?.map((ide) => (
                          <option key={ide.name} value={ide.name!}>
                            {ide.displayName}
                          </option>
                        ))}
                      </Select>
                    </InputGroup>
                    <MenuItem icon={<ArrowPath boxSize={4} />} onClick={onRebuildOpen}>
                      Rebuild
                    </MenuItem>
                    <MenuItem
                      onClick={() => workspace.stop()}
                      icon={<Pause boxSize={5} />}
                      isDisabled={workspace.data.status !== "Running"}>
                      Stop
                    </MenuItem>
                    <MenuItem
                      fontWeight="normal"
                      icon={<Trash boxSize={5} />}
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
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onDeleteClose}>Close</Button>
              <Button
                colorScheme={"red"}
                onClick={async () => {
                  workspace.remove()
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

function getIDEName(ide: TWorkspace["ide"], ides: TIDEs | undefined) {
  const maybeIDE = ides?.find((i) => i.name === ide?.name)

  return maybeIDE?.displayName ?? ide?.name ?? maybeIDE?.name ?? "Unknown"
}
