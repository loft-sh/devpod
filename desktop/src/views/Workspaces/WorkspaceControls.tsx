import { TWorkspaceResult } from "@/contexts/DevPodContext/workspaces/useWorkspace"
import { ChevronRightIcon } from "@chakra-ui/icons"
import {
  Button,
  ButtonGroup,
  HStack,
  Icon,
  IconButton,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Popover,
  PopoverContent,
  PopoverTrigger,
  Portal,
  Text,
  Tooltip,
  useToast,
} from "@chakra-ui/react"
import { useMemo, useCallback, useState } from "react"
import { HiOutlineCode, HiShare } from "react-icons/hi"
import { client } from "@/client"
import { IDEGroup, IDEIcon } from "@/components"
import { TActionID, useProInstances } from "@/contexts"
import {
  ArrowCycle,
  ArrowPath,
  CommandLine,
  Cog,
  Ellipsis,
  Pause,
  Play,
  Stack3D,
  Trash,
} from "@/icons"
import { getIDEDisplayName, useHover } from "@/lib"
import { TIDE, TIDEs, TProInstance, TProvider, TWorkspace, TWorkspaceID } from "@/types"
import { useGroupIDEs } from "@/useIDEs"

type TWorkspaceControlsProps = Readonly<{
  id: TWorkspaceID
  workspace: TWorkspaceResult<TWorkspace>
  provider: TProvider | undefined
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
  onTroubleshootClicked: VoidFunction
  onChangeOptionsClicked?: VoidFunction
}>
export function WorkspaceControls({
  id,
  workspace,
  isLoading,
  ides,
  ideName,
  isIDEFixed,
  provider,
  setIdeName,
  navigateToAction,
  onRebuildClicked,
  onResetClicked,
  onDeleteClicked,
  onStopClicked,
  onLogsClicked,
  onTroubleshootClicked,
  onChangeOptionsClicked,
}: TWorkspaceControlsProps) {
  const [[proInstances]] = useProInstances()
  const proInstance = useMemo<TProInstance | undefined>(() => {
    if (!provider?.isProxyProvider) {
      return undefined
    }

    return proInstances?.find((instance) => instance.provider === provider.config?.name)
  }, [proInstances, provider?.config?.name, provider?.isProxyProvider])
  const { isEnabled: isShareEnabled, onClick: handleShareClicked } = useShareWorkspace(
    workspace.data,
    proInstance
  )

  const handleOpenWithIDEClicked = useCallback(
    (id: TWorkspaceID, ide: TIDE["name"]) => async () => {
      if (!ide) {
        return
      }
      setIdeName(ide)

      const actionID = workspace.start({ id, ideConfig: { name: ide } })
      if (!isIDEFixed) {
        await client.ides.useIDE(ide)
      }
      navigateToAction(actionID)
    },
    [isIDEFixed, setIdeName, workspace, navigateToAction]
  )

  const onIDESelected = useCallback(
    (selectedIDE: TIDE["name"]) => {
      handleOpenWithIDEClicked(id, selectedIDE)()
    },
    [id, handleOpenWithIDEClicked]
  )

  const isOpenDisabled = workspace.data?.status === "Busy"
  const isOpenDisabledReason =
    "Cannot open this workspace because it is busy. If this doesn't change, try to force delete and recreate it."
  const [isStartWithHovering, startWithRef] = useHover()
  const [isPopoverHovering, popoverContentRef] = useHover()

  const [ideGroupHoverState, setIdeGroupHoverState] = useState<{ [key: string]: boolean }>({})

  const setIdeGroupHovered = useCallback(
    (group: string, hovered: boolean) => {
      setIdeGroupHoverState((old) => ({ ...old, [group]: hovered }))
    },
    [setIdeGroupHoverState]
  )

  const ideGroupHovered = useMemo(() => {
    return Object.values(ideGroupHoverState).includes(true)
  }, [ideGroupHoverState])

  const isChangeOptionsEnabled =
    workspace.data?.provider?.options != null && proInstance !== undefined

  const groupedIDEs = useGroupIDEs(ides)

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
                isOpen={isStartWithHovering || isPopoverHovering || ideGroupHovered}
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
                <Portal>
                  <PopoverContent zIndex="popover" width="fit-content" ref={popoverContentRef}>
                    {groupedIDEs?.primary.map((ide) => (
                      <MenuItem
                        isDisabled={isOpenDisabled || isLoading}
                        onClick={handleOpenWithIDEClicked(id, ide.name)}
                        key={ide.name}
                        value={ide.name!}
                        icon={<IDEIcon ide={ide} width={6} height={6} size="sm" />}>
                        {getIDEDisplayName(ide)}
                      </MenuItem>
                    ))}
                    {groupedIDEs?.subMenuGroups.map((group) => (
                      <IDEGroup
                        key={group}
                        ides={groupedIDEs.subMenus[group]}
                        group={group}
                        onHoverChange={setIdeGroupHovered}
                        disabled={isOpenDisabled || isLoading}
                        onItemClick={onIDESelected}
                      />
                    ))}
                  </PopoverContent>
                </Portal>
              </Popover>
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
              {isChangeOptionsEnabled && (
                <MenuItem
                  icon={<Stack3D boxSize={4} />}
                  onClick={onChangeOptionsClicked}
                  isDisabled={isOpenDisabled || isLoading}>
                  Change Options
                </MenuItem>
              )}
              {isShareEnabled && (
                <MenuItem icon={<Icon as={HiShare} boxSize={4} />} onClick={handleShareClicked}>
                  Share
                </MenuItem>
              )}
              <MenuItem
                fontWeight="normal"
                icon={<CommandLine boxSize={4} />}
                onClick={onLogsClicked}>
                Logs
              </MenuItem>
              <MenuItem
                fontWeight="normal"
                icon={<Cog boxSize={4} />}
                onClick={onTroubleshootClicked}>
                Troubleshoot
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

function useShareWorkspace(
  workspace: TWorkspace | undefined,
  proInstance: TProInstance | undefined
) {
  const toast = useToast()

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
    isEnabled: workspace !== undefined && proInstance !== undefined,
    onClick: handleShareClicked,
  }
}
