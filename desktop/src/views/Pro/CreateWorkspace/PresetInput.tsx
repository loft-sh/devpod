import { ManagementV1DevPodWorkspacePreset } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspacePreset"
import {
  Box,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Portal,
  Spinner,
  useToken,
} from "@chakra-ui/react"
import { AiOutlineCodeSandbox } from "react-icons/ai"
import { presetDisplayName } from "@/views/Pro/helpers"

type TPresetInputProps = Readonly<{
  preset?: ManagementV1DevPodWorkspacePreset
  presets?: readonly ManagementV1DevPodWorkspacePreset[]
  setPreset?: (presetId: string | undefined) => void
  loading?: boolean
  isUpdate?: boolean
}>
export function PresetInput({ preset, presets, loading, isUpdate, setPreset }: TPresetInputProps) {
  const primaryColor = useToken("colors", "primary.500")
  const unusedColor = useToken("colors", "divider.main")

  const displayName = presetDisplayName(preset)

  return loading ? (
    <Spinner />
  ) : (
    <Box
      display={"flex"}
      flexDirection={"row"}
      alignItems={"center"}
      gap={3}
      paddingY={"4"}
      paddingX={"6"}
      border={"1px"}
      borderRadius={"4px"}
      transitionProperty={"border-color,color"}
      transitionDuration={"0.3s"}
      color={displayName ? primaryColor : unusedColor}
      borderColor={displayName ? primaryColor : unusedColor}>
      <AiOutlineCodeSandbox opacity={0.7} size={"24"} />
      <Box color={"text.secondary"} fontWeight={displayName ? "semibold" : undefined}>
        {displayName ?? "No preset selected"}
      </Box>

      {!isUpdate && (
        <Box ml={"auto"}>
          <Menu>
            <MenuButton as={Box} cursor={"pointer"} fontWeight={"semibold"} color={primaryColor}>
              Change Preset
            </MenuButton>
            <Portal>
              <MenuList zIndex={"popover"}>
                <MenuItem
                  fontSize={"md"}
                  color={"text.tertiary"}
                  onClick={() => {
                    setPreset?.(undefined)
                  }}>
                  {"No preset"}
                </MenuItem>
                {presets?.map((p, i) => (
                  <MenuItem
                    fontSize={"md"}
                    key={i}
                    onClick={() => {
                      setPreset?.(p.metadata?.name)
                    }}>
                    {presetDisplayName(p) ?? ""}
                  </MenuItem>
                ))}
              </MenuList>
            </Portal>
          </Menu>
        </Box>
      )}
    </Box>
  )
}
