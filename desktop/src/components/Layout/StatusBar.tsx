import { CheckIcon, StarIcon } from "@chakra-ui/icons"
import {
  BoxProps,
  Checkbox,
  HStack,
  Icon,
  IconButton,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Text,
  Tooltip,
  useColorModeValue,
} from "@chakra-ui/react"
import { useMemo } from "react"
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

  return (
    <Menu>
      <MenuButton
        as={IconButton}
        variant="ghost"
        aria-label="zoom"
        rounded="full"
        icon={<Icon boxSize="5" color="iconColor" as={HiMagnifyingGlassPlus} />}
      />
      <MenuList>
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
  const debug = useDebug()

  if (!debug.isEnabled) {
    return null
  }

  const handleMenuItemClicked =
    (option: Parameters<NonNullable<(typeof Debug)["toggle"]>>[0]) => (e: React.MouseEvent) => {
      Debug.toggle?.(option)
      e.stopPropagation()
    }

  return (
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
      </MenuList>
    </Menu>
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
