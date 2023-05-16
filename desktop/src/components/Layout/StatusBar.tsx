import { StarIcon } from "@chakra-ui/icons"
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
} from "@chakra-ui/react"
import { FaBug } from "react-icons/fa"
import { HiDocumentMagnifyingGlass } from "react-icons/hi2"
import { version } from "../../../package.json"
import { client } from "../../client"
import { Debug, useArch, useDebug, usePlatform } from "../../lib"

export function StatusBar(boxProps: BoxProps) {
  const arch = useArch()
  const platform = usePlatform()
  const debug = useDebug()

  return (
    <HStack justify="space-between" paddingX="6" fontSize="sm" zIndex="overlay" {...boxProps}>
      <Text>
        Version {version} | {platform ?? "unknown platform"} | {arch ?? "unknown arch"}
      </Text>
      <HStack>
        <Tooltip label="Loving DevPod? Give us a star on Github">
          <IconButton
            variant="ghost"
            rounded="full"
            icon={<StarIcon color="gray.700" />}
            aria-label="Loving DevPod? Give us a star on Github"
            onClick={() => client.openLink("https://github.com/loft-sh/devpod")}
          />
        </Tooltip>
        <Tooltip label="How to DevPod - Docs">
          <IconButton
            variant="ghost"
            rounded="full"
            icon={<Icon as={HiDocumentMagnifyingGlass} color="gray.700" />}
            aria-label="How to DevPod - Docs"
            onClick={() => client.openLink("https://devpod.sh/docs")}
          />
        </Tooltip>
        <Tooltip label="Report an Issue">
          <IconButton
            variant="ghost"
            rounded="full"
            icon={<Icon as={FaBug} color="gray.700" />}
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
