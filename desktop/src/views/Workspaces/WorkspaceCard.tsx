import { TWorkspaceResult } from "@/contexts/DevPodContext/workspaces/useWorkspace"
import { ChevronRightIcon } from "@chakra-ui/icons"
import {
  Box,
  Button,
  ButtonGroup,
  Card,
  CardHeader,
  Checkbox,
  Heading,
  HStack,
  Icon,
  IconButton,
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
  Text,
  Tooltip,
  useDisclosure,
  useToast,
  VStack,
} from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import dayjs from "dayjs"
import { useCallback, useId, useMemo, useState } from "react"
import { HiClock, HiOutlineCode, HiShare } from "react-icons/hi"
import { useNavigate } from "react-router"
import { client } from "../../client"
import { IconTag, IDEIcon } from "../../components"
import {
  TActionID,
  TActionObj,
  useProInstances,
  useProvider,
  useSettings,
  useWorkspace,
  useWorkspaceActions,
} from "../../contexts"
import {
  ArrowCycle,
  ArrowPath,
  CommandLine,
  Ellipsis,
  Pause,
  Play,
  Stack3D,
  Trash,
} from "../../icons"
import { getIDEDisplayName, useHover } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import { Routes } from "../../routes"
import { TIDE, TIDEs, TProInstance, TWorkspace, TWorkspaceID } from "../../types"
import { useIDEs } from "../../useIDEs"
import { getIDEName, getSourceName } from "./helpers"
import { WorkspaceStatusBadge } from "./WorkspaceStatusBadge"

type TWorkspaceCardProps = Readonly<{
  workspaceID: TWorkspaceID
  isSelected?: boolean
  onSelectionChange?: (isSelected: boolean) => void
}>

export function WorkspaceCard({ workspaceID, isSelected, onSelectionChange }: TWorkspaceCardProps) {
  const settings = useSettings()
  const [forceDelete, setForceDelete] = useState<boolean>(false)
  const navigate = useNavigate()
  const { ides, defaultIDE } = useIDEs()
  const {
    isOpen: isDeleteOpen,
    onOpen: handleDeleteClicked,
    onClose: onDeleteClose,
  } = useDisclosure()
  const {
    isOpen: isRebuildOpen,
    onOpen: handleRebuildClicked,
    onClose: onRebuildClose,
  } = useDisclosure()
  const { isOpen: isResetOpen, onOpen: handleResetClicked, onClose: onResetClose } = useDisclosure()

  const { isOpen: isStopOpen, onOpen: handleStopClicked, onClose: onStopClose } = useDisclosure()
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

  const handleLogsClicked = useCallback(() => {
    let actionID = workspace.current?.id
    if (actionID === undefined) {
      actionID = workspace.checkStatus()
    }

    navigateToAction(actionID)
  }, [navigateToAction, workspace])

  const isLoading = workspace.current?.status === "pending"

  if (workspace.data === undefined) {
    return null
  }

  return (
    <>
      <Card
        key={workspace.data.id}
        direction="row"
        width="full"
        maxWidth="60rem"
        variant="outline"
        backgroundColor={isSelected ? "gray.50" : "transparent"}
        marginBottom="3">
        <WorkspaceCardHeader
          workspace={workspace.data}
          isLoading={isLoading}
          currentAction={workspace.current}
          ideName={ideName}
          isSelected={isSelected}
          onCheckStatusClicked={() => {
            const actionID = workspace.checkStatus()
            navigateToAction(actionID)
          }}
          onSelectionChange={onSelectionChange}
          onActionIndicatorClicked={navigateToAction}>
          <WorkspaceControls
            id={workspace.data.id}
            workspace={workspace}
            isLoading={isLoading}
            isIDEFixed={settings.fixedIDE}
            ides={ides}
            ideName={ideName}
            setIdeName={setIdeName}
            navigateToAction={navigateToAction}
            onRebuildClicked={handleRebuildClicked}
            onResetClicked={handleResetClicked}
            onDeleteClicked={handleDeleteClicked}
            onStopClicked={handleStopClicked}
            onLogsClicked={handleLogsClicked}
          />
        </WorkspaceCardHeader>
      </Card>

      <Modal onClose={onResetClose} isOpen={isResetOpen} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Reset Workspace</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            Reseting the workspace will erase all state saved in the docker container overlay and
            DELETE ALL UNCOMMITTED CODE. This means you might need to reinstall or reconfigure
            certain applications. You will start with a fresh clone of the repository. Are you sure
            you want to rebuild {workspace.data.id}?
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onResetClose}>Close</Button>
              <Button
                colorScheme={"primary"}
                onClick={async () => {
                  const actionID = workspace.reset()
                  onResetClose()
                  navigateToAction(actionID)
                }}>
                Reset
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
  isSelected?: boolean
  onActionIndicatorClicked: (actionID: TActionID | undefined) => void
  onCheckStatusClicked?: VoidFunction
  onSelectionChange?: (isSelected: boolean) => void
  children?: React.ReactNode
}>
function WorkspaceCardHeader({
  workspace,
  isLoading,
  currentAction,
  ideName,
  isSelected,
  onSelectionChange,
  onCheckStatusClicked,
  onActionIndicatorClicked,
  children,
}: TWorkspaceCardHeaderProps) {
  const navigate = useNavigate()
  const checkboxID = useId()
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
    <CardHeader overflow="hidden" w="full">
      <VStack align="start" spacing={0}>
        <HStack w="full">
          <Checkbox
            id={checkboxID}
            paddingRight="2"
            isChecked={isSelected}
            isDisabled={onSelectionChange === undefined}
            onChange={(e) => onSelectionChange?.(e.target.checked)}
          />
          <Heading size="md">
            <HStack alignItems="baseline" justifyContent="space-between">
              <Text
                as="label"
                htmlFor={checkboxID}
                fontWeight="bold"
                maxWidth="23rem"
                overflow="hidden"
                whiteSpace="nowrap"
                textOverflow="ellipsis">
                {id}
              </Text>
              <Box transform="translateY(1px)">
                <WorkspaceStatusBadge
                  status={status}
                  isLoading={isLoading}
                  hasError={hasError}
                  onClick={handleBadgeClicked}
                />
              </Box>
            </HStack>
          </Heading>
          <Box marginLeft="auto">{children}</Box>
        </HStack>
        {source && (
          <Text
            paddingLeft="8"
            fontSize="sm"
            color="gray.500"
            userSelect="auto"
            maxWidth="30rem"
            overflow="hidden"
            whiteSpace="nowrap"
            textOverflow="ellipsis"
            marginTop={-0.5}
            _hover={{ overflow: "visible", cursor: "text" }}>
            {getSourceName(source)}
          </Text>
        )}
      </VStack>

      <HStack rowGap={2} marginTop={4} flexWrap="wrap" alignItems="center" paddingLeft="8">
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

type TWorkspaceControlsProps = Readonly<{
  id: TWorkspaceID
  workspace: TWorkspaceResult
  isIDEFixed: boolean
  isLoading: boolean
  ides: TIDEs | undefined
  ideName: TIDE["name"]
  setIdeName: (ideName: string | undefined) => void
  navigateToAction: (actionID: TActionID | undefined) => void
  onRebuildClicked: VoidFunction
  onResetClicked: VoidFunction
  onDeleteClicked: VoidFunction
  onStopClicked: VoidFunction
  onLogsClicked: VoidFunction
}>
function WorkspaceControls({
  id,
  workspace,
  isLoading,
  ides,
  ideName,
  isIDEFixed,
  setIdeName,
  navigateToAction,
  onRebuildClicked,
  onResetClicked,
  onDeleteClicked,
  onStopClicked,
  onLogsClicked,
}: TWorkspaceControlsProps) {
  const { isEnabled: isShareEnabled, onClick: handleShareClicked } = useShareWorkspace(
    workspace.data
  )

  const handleOpenWithIDEClicked = (id: TWorkspaceID, ide: TIDE["name"]) => async () => {
    if (!ide) {
      return
    }
    setIdeName(ide)

    const actionID = workspace.start({ id, ideConfig: { name: ide } })
    if (!isIDEFixed) {
      await client.ides.useIDE(ide)
    }
    navigateToAction(actionID)
  }
  const isOpenDisabled = workspace.data?.status === "Busy"
  const isOpenDisabledReason =
    "Cannot open this workspace because it is busy. If this doesn't change, try to force delete and recreate it."
  const [isStartWithHovering, startWithRef] = useHover()
  const [isPopoverHovering, popoverContentRef] = useHover()

  return (
    <HStack spacing="2" width="full" justifyContent="end">
      <ButtonGroup isAttached variant="solid-outline">
        <Tooltip label={isOpenDisabled ? isOpenDisabledReason : undefined}>
          <Button
            aria-label="Start workspace"
            leftIcon={<Icon as={HiOutlineCode} boxSize={5} />}
            isDisabled={isOpenDisabled}
            onClick={() => {
              const actionID = workspace.start({
                id,
                ideConfig: { name: ideName ?? ideName ?? null },
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
            colorScheme="gray"
            icon={<Ellipsis transform={"rotate(90deg)"} boxSize={5} />}
          />
          <Portal>
            <MenuList>
              <Popover
                isOpen={isStartWithHovering || isPopoverHovering}
                placement="right"
                offset={[100, 0]}>
                <PopoverTrigger>
                  <MenuItem
                    ref={startWithRef}
                    icon={<Play boxSize={4} />}
                    isDisabled={isOpenDisabled || isLoading}>
                    <HStack width="full" justifyContent="space-between">
                      <Text>Start with</Text>
                      <ChevronRightIcon boxSize={4} />
                    </HStack>
                  </MenuItem>
                </PopoverTrigger>
                <PopoverContent zIndex="popover" width="fit-content" ref={popoverContentRef}>
                  {ides?.map((ide) => (
                    <MenuItem
                      isDisabled={isOpenDisabled || isLoading}
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
                icon={<ArrowPath boxSize={4} />}
                onClick={onRebuildClicked}
                isDisabled={isOpenDisabled || isLoading}>
                Rebuild
              </MenuItem>
              <MenuItem
                icon={<ArrowCycle boxSize={4} />}
                onClick={onResetClicked}
                isDisabled={isOpenDisabled || isLoading}>
                Reset
              </MenuItem>
              {isShareEnabled && (
                <MenuItem icon={<Icon as={HiShare} boxSize={4} />} onClick={handleShareClicked}>
                  Share
                </MenuItem>
              )}
              <MenuItem
                isDisabled={workspace.data?.status !== "Running"}
                onClick={() => {
                  if (workspace.data?.status !== "Running") {
                    onStopClicked()

                    return
                  }

                  workspace.stop()
                }}
                icon={<Pause boxSize={4} />}>
                Stop
              </MenuItem>
              <MenuItem
                fontWeight="normal"
                icon={<CommandLine boxSize={4} />}
                onClick={onLogsClicked}>
                Logs
              </MenuItem>
              <MenuItem
                isDisabled={isOpenDisabled || isLoading}
                fontWeight="normal"
                icon={<Trash boxSize={4} />}
                onClick={onDeleteClicked}>
                Delete
              </MenuItem>
            </MenuList>
          </Portal>
        </Menu>
      </ButtonGroup>
    </HStack>
  )
}

function useShareWorkspace(workspace: TWorkspace | undefined) {
  const toast = useToast()
  const [provider] = useProvider(workspace?.provider?.name)
  const [[proInstances]] = useProInstances()
  const proInstance = useMemo<TProInstance | undefined>(() => {
    if (!provider?.isProxyProvider) {
      return undefined
    }

    return proInstances?.find((instance) => instance.provider === provider.config?.name)
  }, [proInstances, provider?.config?.name, provider?.isProxyProvider])

  const handleShareClicked = useCallback(async () => {
    const devpodProHost = proInstance?.host
    const workspace_id = workspace?.id
    const workspace_uid = workspace?.uid
    if (!devpodProHost || !workspace_id || !workspace_uid) {
      return
    }

    const searchParams = new URLSearchParams()
    searchParams.set("workspace-uid", workspace_uid)
    searchParams.set("workspace-id", workspace_id)
    searchParams.set("devpod-pro-host", devpodProHost)

    const link = `https://devpod.sh/import#${searchParams.toString()}`
    const res = await client.writeToClipboard(link)
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
  }, [proInstance?.host, toast, workspace?.id, workspace?.uid])

  return {
    isEnabled: workspace !== undefined && provider?.isProxyProvider && proInstance !== undefined,
    onClick: handleShareClicked,
  }
}
