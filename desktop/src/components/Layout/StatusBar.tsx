import { CheckIcon, StarIcon } from "@chakra-ui/icons"
import {
  BoxProps,
  Button,
  Checkbox,
  HStack,
  Icon,
  IconButton,
  Input,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalHeader,
  ModalOverlay,
  Text,
  Tooltip,
  useColorModeValue,
  useDisclosure,
} from "@chakra-ui/react"
import { useMemo, useRef } from "react"
import { FaBug } from "react-icons/fa"
import { HiDocumentMagnifyingGlass, HiMagnifyingGlassPlus } from "react-icons/hi2"
import { client } from "../../client"
import { useChangeSettings } from "../../contexts"
import { Debug, useArch, useDebug, usePlatform, useVersion } from "../../lib"

export function StatusBar(boxProps: BoxProps) {
  return (
    <HStack
      justify="space-between"
      paddingX="6"
      fontSize="sm"
      zIndex="overlay"
      {...boxProps}
      color="gray.600"
      _dark={{ color: "gray.400" }}
    />
  )
}

function Version() {
  const version = useVersion()

  return <Text>{version ?? "unknown version"}</Text>
}

function Platform() {
  const platform = usePlatform()

  return <Text>{platform ?? "unknown platform"}</Text>
}

function Arch() {
  const arch = useArch()

  return <Text>{arch ?? "unknown arch"}</Text>
}

function ZoomMenu() {
  const { settings, set } = useChangeSettings()
  const zoomIcon = useMemo(() => {
    return { icon: <CheckIcon /> }
  }, [])
  const textColor = useColorModeValue("gray.900", "gray.300")

  return (
    <Menu>
      <MenuButton
        as={IconButton}
        variant="ghost"
        aria-label="zoom"
        rounded="full"
        icon={<Icon boxSize="5" color="iconColor" as={HiMagnifyingGlassPlus} />}
      />
      <MenuList color={textColor}>
        <MenuItem onClick={() => set("zoom", "sm")} {...(settings.zoom === "sm" ? zoomIcon : {})}>
          Small
        </MenuItem>
        <MenuItem onClick={() => set("zoom", "md")} {...(settings.zoom === "md" ? zoomIcon : {})}>
          Regular
        </MenuItem>
        <MenuItem onClick={() => set("zoom", "lg")} {...(settings.zoom === "lg" ? zoomIcon : {})}>
          Large
        </MenuItem>
        <MenuItem onClick={() => set("zoom", "xl")} {...(settings.zoom === "xl" ? zoomIcon : {})}>
          Extra Large
        </MenuItem>
      </MenuList>
    </Menu>
  )
}

function GitHubStar() {
  const iconColor = useStatusBarIconColor()

  return (
    <Tooltip label="Loving DevPod? Give us a star on Github">
      <IconButton
        variant="ghost"
        rounded="full"
        icon={<StarIcon color={iconColor} />}
        aria-label="Loving DevPod? Give us a star on Github"
        onClick={() => client.open("https://github.com/loft-sh/devpod")}
      />
    </Tooltip>
  )
}

function OSSDocs() {
  const iconColor = useStatusBarIconColor()

  return (
    <Tooltip label="How to DevPod - Docs">
      <IconButton
        variant="ghost"
        rounded="full"
        icon={<Icon as={HiDocumentMagnifyingGlass} color={iconColor} />}
        aria-label="How to DevPod - Docs"
        onClick={() => client.open("https://devpod.sh/docs")}
      />
    </Tooltip>
  )
}

function OSSReportIssue() {
  const iconColor = useStatusBarIconColor()

  return (
    <Tooltip label="Report an Issue">
      <IconButton
        variant="ghost"
        rounded="full"
        icon={<Icon as={FaBug} color={iconColor} />}
        aria-label="Report an Issue"
        onClick={() => client.open("https://github.com/loft-sh/devpod/issues/new/choose")}
      />
    </Tooltip>
  )
}

function DebugMenu() {
  const inputRef = useRef<HTMLInputElement>(null)
  const debug = useDebug()
  const { isOpen, onClose, onOpen: openModal } = useDisclosure()

  if (!debug.isEnabled) {
    return null
  }

  const handleMenuItemClicked =
    (option: Parameters<NonNullable<(typeof Debug)["toggle"]>>[0]) => (e: React.MouseEvent) => {
      Debug.toggle?.(option)
      e.stopPropagation()
    }

  const handleImportLinkClicked = () => {
    const rawLink = inputRef.current?.value
    if (!rawLink) {
      return
    }
    const url = new URL(rawLink.replace(/#/g, "?"))
    const workspaceUID = url.searchParams.get("workspace-uid")
    const workspaceID = url.searchParams.get("workspace-id")
    const host = url.searchParams.get("devpod-pro-host")
    const project = url.searchParams.get("project")
    if (!workspaceUID || !workspaceID || !host || !project) {
      console.error(
        "Some parameters are missing for import",
        url,
        Array.from(url.searchParams.entries())
      )

      return
    }
    client.emitEvent({
      type: "ImportWorkspace",
      workspace_uid: workspaceUID,
      workspace_id: workspaceID,
      devpod_pro_host: host,
      project,
      options: {},
    })
    onClose()
  }

  return (
    <>
      <Menu>
        <MenuButton>Debug</MenuButton>
        <MenuList>
          <MenuItem onClick={handleMenuItemClicked("commands")}>
            <Checkbox isChecked={debug.options.commands} />
            <Text paddingLeft="4">Print command logs</Text>
          </MenuItem>
          <MenuItem onClick={handleMenuItemClicked("actions")}>
            <Checkbox isChecked={debug.options.actions} />
            <Text paddingLeft="4">Print action logs</Text>
          </MenuItem>
          <MenuItem onClick={handleMenuItemClicked("workspaces")}>
            <Checkbox isChecked={debug.options.workspaces} />
            <Text paddingLeft="4">Print workspace logs</Text>
          </MenuItem>
          <MenuItem
            onClick={(e) => {
              client.openDir("AppData")
              e.stopPropagation()
            }}>
            <Text paddingLeft="4">Open app_dir</Text>
          </MenuItem>
          <MenuItem
            onClick={(e) => {
              openModal()
              e.stopPropagation()
            }}>
            <Text paddingLeft="4">Open import tool</Text>
          </MenuItem>
        </MenuList>
      </Menu>
      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalCloseButton />
          <ModalHeader>Import workspace</ModalHeader>
          <ModalBody pb="8">
            <Text mb="4">Paste a platform import link here</Text>
            <Input mb="4" ref={inputRef} type="text" />
            <Button onClick={handleImportLinkClicked}>Import</Button>
          </ModalBody>
        </ModalContent>
      </Modal>
    </>
  )
}

function useStatusBarIconColor() {
  return useColorModeValue("iconColor", "gray.400")
}

StatusBar.Version = Version
StatusBar.Platform = Platform
StatusBar.Arch = Arch
StatusBar.ZoomMenu = ZoomMenu
StatusBar.GitHubStar = GitHubStar
StatusBar.OSSDocs = OSSDocs
StatusBar.OSSReportIssue = OSSReportIssue
StatusBar.DebugMenu = DebugMenu
