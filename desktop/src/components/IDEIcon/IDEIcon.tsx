import {
  Box,
  BoxProps,
  Icon,
  IconProps,
  Image,
  useColorMode,
  useColorModeValue,
  useToken,
} from "@chakra-ui/react"
import { HiBeaker } from "react-icons/hi2"
import { NoneSvg, NoneSvgDark } from "../../images"
import { TIDE } from "../../types"

const SIZES: Record<NonNullable<TIDEIconProps["size"]>, IconProps> = {
  sm: {
    boxSize: 3,
    padding: "1px",
  },
  md: {
    boxSize: 6,
    padding: "3px",
  },
}

type TIDEIconProps = Readonly<{ ide: TIDE; size?: "sm" | "md" }> & BoxProps
export function IDEIcon({ ide, size = "md", ...boxProps }: TIDEIconProps) {
  const experimentalIconSizeProps = SIZES[size]
  const primaryColorDarkToken = useColorModeValue("primary.800", "primary.400")
  const primaryColorDark = useToken("colors", primaryColorDarkToken)
  const primaryColorLightToken = useColorModeValue("primary.400", "primary.800")
  const primaryColorLight = useToken("colors", primaryColorLightToken)
  const backgroundColor = useColorModeValue("white", "gray.700")
  const { colorMode } = useColorMode()

  const experimentalIconStylingProps =
    size === "sm"
      ? {
          color: primaryColorDark,
        }
      : {
          boxShadow: `inset 0px 0px 0px 1px ${primaryColorDark}55`,
          background:
            colorMode === "light"
              ? `linear-gradient(135deg, ${primaryColorLight}55 50%, ${primaryColorDark}55, ${primaryColorDark}88)`
              : `linear-gradient(135deg, ${primaryColorDark}55 50%, ${primaryColorLight}55, ${primaryColorLight}88)`,
          color: `${primaryColorDark}CC`,
        }
  const fallbackIcon = colorMode === "light" ? NoneSvg : NoneSvgDark
  const icon = colorMode === "light" ? ide.icon : ide.iconDark ?? ide.icon

  return (
    <Box width="full" height="full" position="relative">
      <Image src={icon ?? fallbackIcon} {...boxProps} />
      {ide.experimental && (
        <>
          <Box
            position="absolute"
            bottom="0"
            right="0"
            zIndex="docked"
            borderRadius="full"
            boxSize={experimentalIconSizeProps.boxSize}
            backgroundColor={backgroundColor}
          />
          <Icon
            position="absolute"
            bottom="0"
            right="0"
            zIndex="docked"
            borderRadius="full"
            as={HiBeaker}
            {...experimentalIconSizeProps}
            {...experimentalIconStylingProps}
          />
        </>
      )}
    </Box>
  )
}
