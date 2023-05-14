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
import { useCallback } from "react"
import { FaBug } from "react-icons/fa"
import { version } from "../../../package.json"
import { client, DEVPOD_GIT_REPOSITORY } from "../../client"
import { Debug, useArch, useDebug, usePlatform } from "../../lib"

export function StatusBar(boxProps: BoxProps) {
  const arch = useArch()
  const platform = usePlatform()
  const debug = useDebug()

  const handleReportIssueClicked = useCallback(() => {
    const body = encodeURIComponent(`TODO: Describe the Issue here
---
**Version**: ${version}
**Platform**: ${platform ?? "Unknown Platform"}
**Arch**: ${arch ?? "Unknown Arch"}
    `)
    const link = `${DEVPOD_GIT_REPOSITORY}/issues/new?body=${body}`
    client.openLink(link)
  }, [arch, platform])

  return (
    <HStack justify="space-between" paddingX="6" fontSize="sm" zIndex="overlay" {...boxProps}>
      <Text>
        Version {version} | {platform ?? "unknown platform"} | {arch ?? "unknown arch"}
      </Text>
      <HStack>
        <Tooltip label="Report an Issue">
          <IconButton
            variant="ghost"
            rounded="full"
            icon={<Icon as={FaBug} color="gray.700" />}
            aria-label="Report an Issue"
            onClick={handleReportIssueClicked}
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
