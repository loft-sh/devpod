import React from "react"
import { Ripple } from "@/components"
import { TWorkspace } from "@/types"
import { Box, BoxProps, HStack, IconProps, Text, TextProps } from "@chakra-ui/react"
import { useMemo } from "react"

type TWorkspaceStatusBadgeProps = Readonly<{
  status: TWorkspace["status"]
  isLoading: boolean
  hasError: boolean
  showText?: boolean
  onClick?: () => void
}>
export function WorkspaceStatusBadge({
  onClick,
  status,
  hasError,
  isLoading,
  showText = true,
}: TWorkspaceStatusBadgeProps) {
  const badge = useMemo(() => {
    const sharedProps: BoxProps = {
      as: "span",
      borderRadius: "full",
      width: "12px",
      height: "12px",
      borderWidth: "2px",
      zIndex: "1",
    }
    const sharedTextProps: TextProps = {
      fontWeight: "medium",
      fontSize: "12px",
    }
    const rippleProps: IconProps = {
      boxSize: 8,
      position: "absolute",
      left: "-8px",
      zIndex: "0",
    }

    if (hasError || status === "NotFound") {
      return (
        <>
          <Box {...sharedProps} backgroundColor="white" borderColor="red.400" />
          {showText && (
            <Text {...sharedTextProps} color="red.400">
              Error
            </Text>
          )}
        </>
      )
    }

    if (isLoading) {
      return (
        <>
          <Box {...sharedProps} backgroundColor="white" borderColor="yellow.500" />
          <Ripple {...rippleProps} color="yellow.500" />
          {showText && (
            <Text {...sharedTextProps} color="yellow.500">
              Loading
            </Text>
          )}
        </>
      )
    }

    if (status === "Running") {
      return (
        <>
          <Box {...sharedProps} backgroundColor="green.200" borderColor="green.400" />
          {showText && (
            <Text {...sharedTextProps} color="green.400">
              Running
            </Text>
          )}
        </>
      )
    }

    return (
      <>
        <Box {...sharedProps} backgroundColor="purple.200" borderColor="purple.400" zIndex="1" />
        {showText && (
          <Text {...sharedTextProps} color="purple.400">
            {status ?? "Unknown"}
          </Text>
        )}
      </>
    )
  }, [hasError, isLoading, showText, status])

  return (
    <HStack
      cursor={onClick ? "pointer" : "default"}
      onClick={onClick}
      spacing="1"
      position="relative">
      {badge}
    </HStack>
  )
}
