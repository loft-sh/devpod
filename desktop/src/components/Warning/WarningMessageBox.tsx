import { Box, Text, useColorModeValue } from "@chakra-ui/react"
import React from "react"

const SIZES = {
  sm: {
    fontSize: "sm",
  },
  md: {
    fontSize: "md",
  },
}
const VARIANTS = {
  solid: {
    color: { light: "orange.700", dark: "orange.800" },
  },
  ghost: {
    color: { light: "orange.400", dark: "orange.300" },
  },
}
type TWarningMessageBoxProps = Readonly<{
  warning: React.ReactNode
  size?: keyof typeof SIZES
  variant?: "solid" | "ghost"
}>
export function WarningMessageBox({
  warning,
  size = "md",
  variant = "solid",
}: TWarningMessageBoxProps) {
  const { color } = VARIANTS[variant]
  const backgroundColor = useColorModeValue("orange.100", "orange.200")
  const textColor = useColorModeValue(color.light, color.dark)
  const { fontSize } = SIZES[size]

  return (
    <Box
      {...(variant === "solid"
        ? {
            backgroundColor,
            marginTop: "4",
            padding: "4",
            borderRadius: "md",
          }
        : {})}
      userSelect="text"
      display="inline-block">
      <Text color={textColor} fontSize={fontSize}>
        {warning}
      </Text>
    </Box>
  )
}
