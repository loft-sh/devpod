import { Box, BoxProps, Icon, IconProps, Image, useToken } from "@chakra-ui/react"
import { HiBeaker } from "react-icons/hi2"
import { NoneSvg } from "../../images"
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
  const primaryColorDark = useToken("colors", "primary.800")
  const primaryColorLight = useToken("colors", "primary.400")

  const experimentalIconStylingProps =
    size === "sm"
      ? {
          color: primaryColorDark,
        }
      : {
          boxShadow: `inset 0px 0px 0px 1px ${primaryColorDark}55`,
          background: `linear-gradient(135deg, ${primaryColorLight}55 50%, ${primaryColorDark}55, ${primaryColorDark}88)`,
          color: `${primaryColorDark}CC`,
        }

  return (
    <Box width="full" height="full" position="relative">
      <Image src={ide.icon ?? NoneSvg} {...boxProps} />
      {ide.experimental && (
        <>
          <Box
            position="absolute"
            bottom="0"
            right="0"
            zIndex="docked"
            borderRadius="full"
            boxSize={experimentalIconSizeProps.boxSize}
            backgroundColor="white"
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
