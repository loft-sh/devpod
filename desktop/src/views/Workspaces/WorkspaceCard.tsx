import { ChevronRightIcon } from "@chakra-ui/icons"
import {
  Box,
  BoxProps,
  Button,
  ButtonGroup,
  Card,
  CardFooter,
  CardHeader,
  Checkbox,
  Heading,
  HStack,
  Icon,
  IconButton,
  IconProps,
  Image,
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
  Popover,
  PopoverContent,
  PopoverTrigger,
  Portal,
  Stack,
  Text,
  TextProps,
  Tooltip,
  useDisclosure,
  useToast,
  VStack,
} from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import dayjs from "dayjs"
import { useCallback, useMemo, useState } from "react"
import { HiClock, HiOutlineCode, HiShare } from "react-icons/hi"
import { useNavigate } from "react-router"
import { client } from "../../client"
import { IconTag, Ripple, IDEIcon } from "../../components"
import {
  TActionID,
  TActionObj,
  useSettings,
  useWorkspace,
  useWorkspaceActions,
} from "../../contexts"
import { ArrowPath, Ellipsis, Pause, Play, Stack3D, Trash } from "../../icons"
import { NoWorkspaceImageSvg } from "../../images"
import { exists, getIDEDisplayName, useHover } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import { Routes } from "../../routes"
import { TIDE, TWorkspace, TWorkspaceID } from "../../types"
import { getIDEName, getSourceName } from "./helpers"
import { useIDEs } from "../../useIDEs"

type TWorkspaceCardProps = Readonly<{
  workspaceID: TWorkspaceID
  onSelectionChange?: (isSelected: boolean) => void
}>

export function WorkspaceCard({ workspaceID, onSelectionChange }: TWorkspaceCardProps) {
  const [forceDelete, setForceDelete] = useState<boolean>(false)
  const navigate = useNavigate()
  const toast = useToast()
  const settings = useSettings()
  const { ides, defaultIDE } = useIDEs()
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()
  const { isOpen: isRebuildOpen, onOpen: onRebuildOpen, onClose: onRebuildClose } = useDisclosure()
  const { isOpen: isStopOpen, onOpen: onStopOpen, onClose: onStopClose } = useDisclosure()
  const workspace = useWorkspace(workspaceID)
  const [ideName, setIdeName] = useState<string | undefined>(() => {
    if (settings.fixedIDE && defaultIDE?.name) {
      return defaultIDE.name
    }

    return workspace.data?.ide?.name ?? undefined
  })

  const navigateToAction = useCallback(
    (actionID: TActionID | undefined) => {
      if (actionID !== undefined && actionID !== "") {
        navigate(Routes.toAction(actionID))
      }
    },
    [navigate]
  )

  const handleOpenWithIDEClicked = useCallback(
    (id: TWorkspaceID, ide: TIDE["name"]) => async () => {
      if (!ide) {
        return
      }
      setIdeName(ide)

      const actionID = workspace.start({ id, ideConfig: { name: ide } })
      if (!settings.fixedIDE) {
        await client.ides.useIDE(ide)
      }
      navigateToAction(actionID)
    },
    [workspace, settings.fixedIDE, navigateToAction]
  )

  const handleShareClicked = useCallback(
    (id: TWorkspaceID) => async () => {
      if (workspace.data === undefined) {
        return
      }

      if (!exists(workspace.data.source)) {
        return
      }

      const source = encodeURIComponent(getSourceName(workspace.data.source))
      const workspaceID = encodeURIComponent(id)
      let devpodLink = `https://devpod.sh/open#${source}&workspace=${workspaceID}`
      const maybeProviderName = workspace.data.provider?.name
      if (exists(maybeProviderName)) {
        devpodLink = devpodLink.concat(`&provider=${encodeURIComponent(maybeProviderName)}`)
      }
      const maybeIDEName = workspace.data.ide?.name
      if (exists(maybeIDEName)) {
        devpodLink = devpodLink.concat(`&ide=${encodeURIComponent(maybeIDEName)}`)
      }

      const res = await client.writeToClipboard(devpodLink)
      if (!res.ok) {
        toast({
          title: "Failed to share workspace",
          description: res.val.message,
          status: "error",
          duration: 5_000,
          isClosable: true,
        })

        return
      }

      toast({
        title: "Copied workspace link to clipboard",
        status: "success",
        duration: 5_000,
        isClosable: true,
      })
    },
    [toast, workspace.data]
  )

  const isLoading = useMemo(() => {
    if (workspace.current?.status === "pending") {
      return true
    }

    return false
  }, [workspace])

  const isOpenDisabled = workspace.data?.status === "Busy"
  const isOpenDisabledReason =
    "Cannot open this workspace because it is busy. If this doesn't change, try to force delete and recreate it."
  const [isStartWithHovering, startWithRef] = useHover()
  const [isPopoverHovering, popoverContentRef] = useHover()

  if (workspace.data === undefined) {
    return null
  }

  const { id, picture, ide, status } = workspace.data

  return (
    <>
      <Card key={id} direction="row" width="full" maxWidth="60rem" variant="outline" maxHeight="48">
        <Image
          loading="lazy"
          objectFit="contain"
          width="18.75rem"
          height="12rem"
          style={{ aspectRatio: "2 / 1" }}
          src={picture ?? NoWorkspaceImageSvg}
          fallbackSrc={NoWorkspaceImageSvg}
          alt="Project Image"
        />
        <Stack width="full" justifyContent={"space-between"}>
          <WorkspaceCardHeader
            workspace={workspace.data}
            isLoading={isLoading}
            currentAction={workspace.current}
            ideName={ideName}
            onCheckStatusClicked={() => {
              const actionID = workspace.checkStatus()
              navigateToAction(actionID)
            }}
            onSelectionChange={onSelectionChange}
            onActionIndicatorClicked={navigateToAction}
          />

          <CardFooter padding="none" paddingBottom={4}>
            <HStack spacing="2" width="full" justifyContent="end" paddingRight={"10px"}>
              <ButtonGroup isAttached variant="solid-outline">
                <Tooltip label={isOpenDisabled ? isOpenDisabledReason : undefined}>
                  <Button
                    aria-label="Start workspace"
                    leftIcon={<Icon as={HiOutlineCode} boxSize={5} />}
                    isDisabled={isOpenDisabled}
                    onClick={() => {
                      const actionID = workspace.start({
                        id,
                        ideConfig: { name: ideName ?? ide?.name ?? null },
                      })
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
                    // variant="ghost"
                    colorScheme="gray"
                    isDisabled={isLoading}
                    icon={<Ellipsis transform={"rotate(90deg)"} boxSize={5} />}
                  />
                  <Portal>
                    <MenuList>
                      <Popover
                        isOpen={isStartWithHovering || isPopoverHovering}
                        placement="right"
                        offset={[100, 0]}>
                        <PopoverTrigger>
                          <MenuItem ref={startWithRef} icon={<Play boxSize={4} />}>
                            <HStack width="full" justifyContent="space-between">
                              <Text>Start with</Text>
                              <ChevronRightIcon boxSize={4} />
                            </HStack>
                          </MenuItem>
                        </PopoverTrigger>
                        <PopoverContent
                          zIndex="popover"
                          width="fit-content"
                          ref={popoverContentRef}>
                          {ides?.map((ide) => (
                            <MenuItem
                              onClick={handleOpenWithIDEClicked(id, ide.name)}
                              key={ide.name}
                              value={ide.name!}
                              icon={<IDEIcon ide={ide} width={6} height={6} size="sm" />}>
                              {getIDEDisplayName(ide)}
                            </MenuItem>
                          ))}
                        </PopoverContent>
                      </Popover>
                      <MenuItem
                        icon={<Icon as={HiShare} boxSize={4} />}
                        onClick={handleShareClicked(id)}>
                        Share Configuration
                      </MenuItem>
                      <MenuItem icon={<ArrowPath boxSize={4} />} onClick={onRebuildOpen}>
                        Rebuild
                      </MenuItem>
                      <MenuItem
                        isDisabled={status !== "Running"}
                        onClick={() => {
                          if (status !== "Running") {
                            onStopOpen()

                            return
                          }

                          workspace.stop()
                        }}
                        icon={<Pause boxSize={4} />}>
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
              </ButtonGroup>
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
            <Box marginTop={"2.5"}>
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

      <Modal onClose={onStopClose} isOpen={isStopOpen} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Stop Workspace</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            Stopping the workspace while it&apos;s not running may leave it in a corrupted state. Do
            you want to stop it regardless?
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onStopClose}>Close</Button>
              <Button
                colorScheme={"red"}
                onClick={() => {
                  workspace.stop()
                  onStopClose()
                }}>
                Stop
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
  currentAction: TActionObj | undefined
  ideName: string | undefined
  onActionIndicatorClicked: (actionID: TActionID | undefined) => void
  onCheckStatusClicked?: VoidFunction
  onSelectionChange?: (isSelected: boolean) => void
}>
function WorkspaceCardHeader({
  workspace,
  isLoading,
  currentAction,
  ideName,
  onSelectionChange,
  onCheckStatusClicked,
  onActionIndicatorClicked,
}: TWorkspaceCardHeaderProps) {
  const navigate = useNavigate()
  const { id, status, provider, ide, lastUsed, source } = workspace
  const workspaceActions = useWorkspaceActions(id)

  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })

  const hasError = useMemo<boolean>(() => {
    if (!workspaceActions?.length || workspaceActions[0]?.status !== "error") {
      return false
    }

    return true
  }, [workspaceActions])

  const handleBadgeClicked = useMemo(() => {
    if (currentAction !== undefined) {
      return () => onActionIndicatorClicked(currentAction.id)
    }

    if (status === undefined || status === "NotFound") {
      return () => onCheckStatusClicked?.()
    }

    const maybeLastAction = workspaceActions?.[0]
    if (maybeLastAction) {
      return () => onActionIndicatorClicked(maybeLastAction.id)
    }

    return undefined
  }, [currentAction, onActionIndicatorClicked, onCheckStatusClicked, status, workspaceActions])

  const ideDisplayName =
    ideName !== undefined
      ? getIDEName({ name: ideName }, idesQuery.data)
      : getIDEName(ide, idesQuery.data)

  return (
    <CardHeader display="flex" flexDirection="column" overflow="hidden">
      <VStack align="start" spacing={0}>
        <HStack justifyContent="space-between" maxWidth="30rem">
          <Heading size="md">
            <HStack alignItems="center">
              <Text
                fontWeight="bold"
                maxWidth="23rem"
                overflow="hidden"
                whiteSpace="nowrap"
                textOverflow="ellipsis">
                {id}
              </Text>
              <WorkspaceStatusBadge
                status={status}
                isLoading={isLoading}
                hasError={hasError}
                onClick={handleBadgeClicked}
              />
            </HStack>
          </Heading>
          {onSelectionChange !== undefined && (
            <Checkbox onChange={(e) => onSelectionChange(e.target.checked)} />
          )}
        </HStack>
        {source && (
          <Text
            fontSize="sm"
            color="gray.500"
            userSelect="auto"
            maxWidth="30rem"
            overflow="hidden"
            whiteSpace="nowrap"
            textOverflow="ellipsis"
            _hover={{ overflow: "visible", cursor: "text" }}>
            {getSourceName(source)}
          </Text>
        )}
      </VStack>

      <HStack rowGap={2} marginTop={4} flexWrap="wrap" alignItems="center">
        <IconTag
          icon={<Stack3D />}
          label={provider?.name ?? "No provider"}
          infoText={provider?.name ? `Uses provider ${provider.name}` : undefined}
          onClick={() => {
            if (!provider?.name) {
              return
            }

            navigate(Routes.toProvider(provider.name))
          }}
        />
        <IconTag
          icon={<Icon as={HiOutlineCode} />}
          label={ideDisplayName}
          infoText={`Will be opened in ${ideDisplayName}`}
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

type TWorkspaceStatusBadgeProps = Readonly<{
  status: TWorkspace["status"]
  isLoading: boolean
  hasError: boolean
  onClick?: () => void
}>
function WorkspaceStatusBadge({
  onClick,
  status,
  hasError,
  isLoading,
}: TWorkspaceStatusBadgeProps) {
  const badge = useMemo(() => {
    const sharedProps: BoxProps = {
      as: "span",
      borderRadius: "full",
      width: "12px",
      height: "12px",
      borderWidth: "2px",
      zIndex: "1",
    }
    const sharedTextProps: TextProps = {
      fontWeight: "medium",
      fontSize: "sm",
    }
    const rippleProps: IconProps = {
      boxSize: 8,
      position: "absolute",
      left: "-8px",
      zIndex: "0",
    }

    if (hasError) {
      return (
        <>
          <Box {...sharedProps} backgroundColor="white" borderColor="red.400" />
          <Text {...sharedTextProps} color="red.400">
            Error
          </Text>
        </>
      )
    }

    if (isLoading) {
      return (
        <>
          <Box {...sharedProps} backgroundColor="white" borderColor="yellow.500" />
          <Ripple {...rippleProps} color="yellow.500" />
          <Text {...sharedTextProps} color="yellow.500">
            Loading
          </Text>
        </>
      )
    }

    if (status === "Running") {
      return (
        <>
          <Box {...sharedProps} backgroundColor="green.200" borderColor="green.400" />
          <Text {...sharedTextProps} color="green.400">
            Running
          </Text>
        </>
      )
    }

    return (
      <>
        <Box {...sharedProps} backgroundColor="purple.200" borderColor="purple.400" zIndex="1" />
        <Text {...sharedTextProps} color="purple.400">
          {status ?? "Unknown"}
        </Text>
      </>
    )
  }, [hasError, isLoading, status])

  return (
    <HStack
      cursor={onClick ? "pointer" : "default"}
      onClick={onClick}
      spacing="1"
      position="relative">
      {badge}
    </HStack>
  )
}
