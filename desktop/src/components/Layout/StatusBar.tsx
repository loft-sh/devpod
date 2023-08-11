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
  const arch = useArch()
  const platform = usePlatform()
  const version = useVersion()
  const debug = useDebug()
  const { settings, set } = useChangeSettings()
  const iconColor = useColorModeValue("iconColor", "gray.400")

  const zoomIcon = useMemo(() => {
    return { icon: <CheckIcon /> }
  }, [])

  return (
    <HStack justify="space-between" paddingX="6" fontSize="sm" zIndex="overlay" {...boxProps}>
      <Text>
        Version {version} | {platform ?? "unknown platform"} | {arch ?? "unknown arch"}
      </Text>
      <HStack>
        <Menu>
          <MenuButton
            as={IconButton}
            variant="ghost"
            aria-label="zoom"
            rounded="full"
            icon={<Icon boxSize="5" color="iconColor" as={HiMagnifyingGlassPlus} />}
          />
          <MenuList>
            <MenuItem
              onClick={() => set("zoom", "sm")}
              {...(settings.zoom === "sm" ? zoomIcon : {})}>
              Small
            </MenuItem>
            <MenuItem
              onClick={() => set("zoom", "md")}
              {...(settings.zoom === "md" ? zoomIcon : {})}>
              Regular
            </MenuItem>
            <MenuItem
              onClick={() => set("zoom", "lg")}
              {...(settings.zoom === "lg" ? zoomIcon : {})}>
              Large
            </MenuItem>
            <MenuItem
              onClick={() => set("zoom", "xl")}
              {...(settings.zoom === "xl" ? zoomIcon : {})}>
              Extra Large
            </MenuItem>
          </MenuList>
        </Menu>

        <Tooltip label="Loving DevPod? Give us a star on Github">
          <IconButton
            variant="ghost"
            rounded="full"
            icon={<StarIcon color={iconColor} />}
            aria-label="Loving DevPod? Give us a star on Github"
            onClick={() => client.openLink("https://github.com/loft-sh/devpod")}
          />
        </Tooltip>

        <Tooltip label="How to DevPod - Docs">
          <IconButton
            variant="ghost"
            rounded="full"
            icon={<Icon as={HiDocumentMagnifyingGlass} color={iconColor} />}
            aria-label="How to DevPod - Docs"
            onClick={() => client.openLink("https://devpod.sh/docs")}
          />
        </Tooltip>

        <Tooltip label="Report an Issue">
          <IconButton
            variant="ghost"
            rounded="full"
            icon={<Icon as={FaBug} color={iconColor} />}
            aria-label="Report an Issue"
            onClick={() => client.openLink("https://github.com/loft-sh/devpod/issues/new/choose")}
          />
        </Tooltip>

        {debug.isEnabled && (
          <Menu>
            <MenuButton>Debug</MenuButton>
            <MenuList>
              <MenuItem onClick={() => Debug.toggle?.("commands")}>
                <Checkbox isChecked={debug.options.commands} />
                <Text paddingLeft="4">Print command logs</Text>
              </MenuItem>
              <MenuItem onClick={() => Debug.toggle?.("actions")}>
                <Checkbox isChecked={debug.options.actions} />
                <Text paddingLeft="4">Print action logs</Text>
              </MenuItem>
              <MenuItem onClick={() => Debug.toggle?.("workspaces")}>
                <Checkbox isChecked={debug.options.workspaces} />
                <Text paddingLeft="4">Print workspace logs</Text>
              </MenuItem>
              <MenuItem onClick={() => client.openDir("AppData")}>
                <Text paddingLeft="4">Open app_dir</Text>
              </MenuItem>
            </MenuList>
          </Menu>
        )}
      </HStack>
    </HStack>
  )
}
