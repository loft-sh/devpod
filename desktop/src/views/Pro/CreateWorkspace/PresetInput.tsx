import { useBorderColor } from "@/Theme"
import { presetDisplayName } from "@/views/Pro/helpers"
import {
  Box,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Portal,
  Spinner,
  useColorModeValue,
} from "@chakra-ui/react"
import { ManagementV1DevPodWorkspacePreset } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspacePreset"
import { AiOutlineCodeSandbox } from "react-icons/ai"

type TPresetInputProps = Readonly<{
  preset?: ManagementV1DevPodWorkspacePreset
  presets?: readonly ManagementV1DevPodWorkspacePreset[]
  setPreset?: (presetId: string | undefined) => void
  loading?: boolean
  isUpdate?: boolean
}>
export function PresetInput({ preset, presets, loading, isUpdate, setPreset }: TPresetInputProps) {
  const selectedColor = useColorModeValue("primary.500", "primary.300")
  const unusedColor = useBorderColor()
  const textColor = useColorModeValue("gray.800", "gray.200")

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
      color={displayName ? selectedColor : unusedColor}
      borderColor={displayName ? selectedColor : unusedColor}>
      <AiOutlineCodeSandbox opacity={0.7} size={"24"} />
      <Box color={textColor} fontWeight={displayName ? "semibold" : undefined}>
        {displayName ?? "No preset selected"}
      </Box>

      {!isUpdate && (
        <Box ml={"auto"}>
          <Menu>
            <MenuButton
              as={Box}
              cursor={"pointer"}
              fontWeight={"semibold"}
              color={"primary.500"}
              _dark={{ color: "primary.300" }}>
              Change Preset
            </MenuButton>
            <Portal>
              <MenuList zIndex={"popover"}>
                <MenuItem
                  fontSize={"md"}
                  color={"gray.500"}
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
