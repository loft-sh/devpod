import {
  Checkbox,
  HStack,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Text,
  useColorModeValue,
} from "@chakra-ui/react"
import { version } from "../../../package.json"
import { client } from "../../client"
import { Debug, useArch, useDebug, usePlatform } from "../../lib"

export function StatusBar() {
  const arch = useArch()
  const platform = usePlatform()
  const debug = useDebug()
  const statusBarBackgroundColor = useColorModeValue("gray.300", "gray.600")

  return (
    <HStack
      justify="space-between"
      paddingX="6"
      position="fixed"
      bottom="0"
      backgroundColor={statusBarBackgroundColor}
      width="full"
      fontSize="sm"
      zIndex="1">
      <Text>
        Version {version} | {platform ?? "unknown platform"} | {arch ?? "unknown arch"}
      </Text>
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
  )
}
