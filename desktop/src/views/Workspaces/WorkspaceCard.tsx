import {
  Button,
  Card,
  Image,
  CardBody,
  CardFooter,
  CardHeader,
  Checkbox,
  Heading,
  Stack,
  Text,
  HStack,
  Box,
  IconButton,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverArrow,
  VStack,
  Tooltip,
  useDisclosure,
  Modal,
  ModalOverlay,
  ModalHeader,
  ModalContent,
  ModalCloseButton,
  ModalBody,
  ModalFooter,
  Icon,
  Select,
} from "@chakra-ui/react"
import { useWorkspace } from "../../contexts"
import { TWorkspaceID } from "../../types"
import { Pause, Trash } from "../../icons"
import CodeImage from "../../images/code.jpg"
import { Ellipsis } from "../../icons/Ellipsis"
import dayjs from "dayjs"
import { GrCode, GrRefresh } from "react-icons/gr"
import { useQuery } from "@tanstack/react-query"
import { QueryKeys } from "../../queryKeys"
import { client } from "../../client"
import { useState } from "react"

type TWorkspaceCardProps = Readonly<{
  workspaceID: TWorkspaceID
  onSelectionChange?: (isSelected: boolean) => void
}>

export function WorkspaceCard({ workspaceID, onSelectionChange }: TWorkspaceCardProps) {
  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()
  const { isOpen: isRebuildOpen, onOpen: onRebuildOpen, onClose: onRebuildClose } = useDisclosure()
  const {
    isOpen: isOpenWithOpen,
    onOpen: onOpenWithOpen,
    onClose: onOpenWithClose,
  } = useDisclosure()
  const workspace = useWorkspace(workspaceID)
  const [ideName, setIdeName] = useState<string | undefined>(workspace.data?.ide?.name || undefined)
  if (workspace.data === undefined) {
    return null
  }

  const { id, provider, status, ide } = workspace.data

  return (
    <>
      <Card
        key={id}
        direction={{ base: "row", sm: "row" }}
        width={"100%"}
        maxWidth={"600px"}
        overflow="hidden"
        variant="outline">
        <Image
          objectFit="cover"
          maxW={{ base: "100%", sm: "200px" }}
          src={CodeImage}
          alt="Project Image"
        />

        <Stack>
          <CardHeader display="flex" width="full" justifyContent="space-between">
            <Heading size="md">
              <HStack>
                <Text fontWeight="bold">{id}</Text>
                <Tooltip label={`Workspace is ${status}`}>
                  <Box
                    as={"span"}
                    display={"inline-block"}
                    backgroundColor={status === "Running" ? "green" : "orange"}
                    borderRadius={"20px"}
                    width={"10px"}
                    height={"10px"}
                    position={"relative"}
                    top={"1px"}
                  />
                </Tooltip>
              </HStack>
            </Heading>
            {onSelectionChange !== undefined && (
              <Checkbox onChange={(e) => onSelectionChange(e.target.checked)} />
            )}
          </CardHeader>
          <CardBody>
            {provider?.name && <Text>Provider: {provider.name}</Text>}
            {workspace.data.ide?.name && <Text>IDE: {workspace.data.ide?.name}</Text>}
            <Text>Last Used: {dayjs(new Date(workspace.data.lastUsed)).fromNow()}</Text>
          </CardBody>
          <CardFooter>
            {workspace.data.status !== "Busy" && workspace.data.status !== "NotFound" && (
              <HStack spacing={"2"}>
                <Button
                  colorScheme="primary"
                  onClick={() => workspace.start({ id, ideConfig: ide })}
                  isLoading={
                    workspace.current?.name === "start" && workspace.current.status === "pending"
                  }>
                  {workspace.data.status === "Stopped" ? "Start" : "Open"}
                </Button>
                {workspace.data.status === "Running" && (
                  <Tooltip label={"Stop workspace"}>
                    <IconButton
                      aria-label="Stop workspace"
                      variant="ghost"
                      colorScheme="gray"
                      onClick={() => workspace.stop()}
                      icon={<Pause width={"16px"} />}
                      isLoading={
                        workspace.current?.name === "stop" && workspace.current.status === "pending"
                      }
                    />
                  </Tooltip>
                )}
                <Tooltip label={`Delete workspace`}>
                  <IconButton
                    aria-label="Delete workspace"
                    variant="ghost"
                    colorScheme="gray"
                    icon={<Trash width={"16px"} />}
                    onClick={() => onDeleteOpen()}
                    isLoading={
                      workspace.current?.name === "remove" && workspace.current.status === "pending"
                    }
                  />
                </Tooltip>
                {workspace.data.status === "Running" && (
                  <Popover trigger={"hover"}>
                    <PopoverTrigger>
                      <IconButton
                        aria-label="More actions"
                        variant="ghost"
                        colorScheme="gray"
                        icon={<Ellipsis width={"16px"} />}
                        onClick={() => {}}
                      />
                    </PopoverTrigger>
                    <PopoverContent>
                      <PopoverArrow />
                      <Box padding={"10px"}>
                        <VStack>
                          <Button
                            leftIcon={<Icon as={GrCode} />}
                            width={"150px"}
                            onClick={() => onOpenWithOpen()}>
                            Open with...
                          </Button>
                          <Button
                            leftIcon={<Icon as={GrRefresh} />}
                            width={"150px"}
                            onClick={() => onRebuildOpen()}>
                            Rebuild
                          </Button>
                        </VStack>
                      </Box>
                    </PopoverContent>
                  </Popover>
                )}
              </HStack>
            )}
          </CardFooter>
        </Stack>
      </Card>
      <Modal onClose={onOpenWithClose} isOpen={isOpenWithOpen} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Open Workspace With IDE</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Heading as="h5" size="s" marginBottom={"10px"}>
              Select IDE:
            </Heading>
            <Select onChange={(e) => setIdeName(e.target.value)} value={ideName}>
              {idesQuery.data?.map((ide) => (
                <option key={ide.name} value={ide.name!}>
                  {ide.displayName}
                </option>
              ))}
            </Select>
            <Text fontSize={"12px"} marginTop={"7px"}>
              Devpod will open this workspace with the selected IDE.
            </Text>
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onOpenWithClose}>Close</Button>
              <Button
                colorScheme={"primary"}
                onClick={async () => {
                  if (!ideName) {
                    return
                  }

                  workspace.start({ id, ideConfig: { name: ideName } })
                  onOpenWithClose()
                }}>
                Open
              </Button>
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>
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
                  workspace.rebuild()
                  onRebuildClose()
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
