import { BoxProps, Card, Image, Text, Tooltip, useColorModeValue, useToken } from "@chakra-ui/react"
import React, { useId } from "react"

type TExampleCardProps = {
  name: string
  image?: string
  size?: keyof typeof sizes

  isSelected?: boolean
  isDisabled?: boolean
  onClick?: () => void
}

const sizes: Record<"sm" | "md" | "lg", BoxProps["width"]> = {
  sm: "12",
  md: "20",
  lg: "24",
} as const

export function ExampleCard({
  name,
  image,
  isSelected,
  isDisabled,
  size = "lg",
  onClick,
}: TExampleCardProps) {
  const hoverBackgroundColor = useColorModeValue("gray.50", "gray.800")
  const primaryColorLight = useToken("colors", "primary.400")
  const primaryColorDark = useToken("colors", "primary.800")

  const selectedProps = isSelected
    ? {
        _before: {
          content: '""',
          position: "absolute",
          top: 0,
          bottom: 0,
          left: 0,
          right: 0,
          background: `linear-gradient(135deg, ${primaryColorLight}55 30%, ${primaryColorDark}55, ${primaryColorDark}88)`,
          opacity: 0.7,
          width: "full",
          height: "full",
        },
        boxShadow: `inset 0px 0px 0px 1px ${primaryColorDark}55`,
      }
    : {}

  const disabledProps = isDisabled ? { filter: "grayscale(100%)", cursor: "not-allowed" } : {}

  return (
    <Tooltip textTransform={"capitalize"} label={name} isDisabled={size === "lg"}>
      <Card
        variant="unstyled"
        width={sizes[size]}
        height={sizes[size]}
        alignItems="center"
        display="flex"
        justifyContent="center"
        cursor="pointer"
        boxSizing="border-box"
        position="relative"
        backgroundColor="transparent"
        overflow="hidden"
        _hover={{ backgroundColor: isDisabled || isSelected ? undefined : hoverBackgroundColor }}
        {...(onClick && !isDisabled && !isSelected ? { onClick } : {})}
        {...selectedProps}
        {...disabledProps}>
        <Image objectFit="fill" overflow="hidden" zIndex="1" src={image} />
        {size === "lg" && (
          <Text
            paddingBottom="1"
            fontSize="11px"
            color="gray.500"
            fontWeight="medium"
            overflow="hidden"
            maxWidth={sizes[size]}
            textOverflow="ellipsis"
            whiteSpace="nowrap"
            textTransform={"capitalize"}>
            {name}
          </Text>
        )}
      </Card>
    </Tooltip>
  )
}
// <AnimatePresence>
//   {isSelected && (
//     <Box
//       as={motion.div}
//       initial={{ opacity: 0 }}
//       animate={{ opacity: 1 }}
//       exit={{ opacity: 0 }}
//       position="absolute"
//       width="full"
//       height="3"
//       bottom={"17px"}>
//       <Box as={Glow} width="full" />
//     </Box>
//   )}
// </AnimatePresence>

function Glow() {
  const id = useId()

  return (
    <svg viewBox="0 0 53 16" width="100%">
      <mask
        id={`${id}-b`}
        width="53"
        height="16"
        x="0"
        y="0"
        maskUnits="userSpaceOnUse"
        style={{ maskType: "alpha" }}>
        <path fill={`url(#${id}-a)`} d="M0 0h53v16H0z" />
      </mask>
      <g mask={`url(#${id}-b)`}>
        <path fill={`url(#${id}-c)`} d="M1 13.077h51V20H1z" />
      </g>
      <mask
        id={`${id}-e`}
        width="53"
        height="3"
        x="0"
        y="11"
        maskUnits="userSpaceOnUse"
        style={{ maskType: "alpha" }}>
        <path fill={`url(#${id}-d)`} d="M0 11h53v3H0z" />
      </mask>
      <g mask={`url(#${id}-e)`}>
        <path fill={`url(#${id}-f)`} d="M1 13.077h51V20H1z" />
      </g>
      <defs>
        <radialGradient
          id={`${id}-a`}
          cx="0"
          cy="0"
          r="1"
          gradientTransform="matrix(0 8 -26.5 0 26.5 8)"
          gradientUnits="userSpaceOnUse">
          <stop stopColor="#C6C6C6" />
          <stop offset="1" stopColor="#D9D9D9" stopOpacity="0" />
        </radialGradient>
        <radialGradient
          id={`${id}-d`}
          cx="0"
          cy="0"
          r="1"
          gradientTransform="matrix(0 1.5 -26.5 0 26.5 12.5)"
          gradientUnits="userSpaceOnUse">
          <stop offset=".635" />
          <stop offset=".906" stopColor="#D9D9D9" stopOpacity="0" />
        </radialGradient>
        <linearGradient
          id={`${id}-c`}
          x1="10.597"
          x2="43.226"
          y1="16.538"
          y2="16.821"
          gradientUnits="userSpaceOnUse">
          <stop stopColor="#FA78C6" />
          <stop offset="1" stopColor="#CA60FF" />
        </linearGradient>
        <linearGradient
          id={`${id}-f`}
          x1="10.597"
          x2="43.226"
          y1="16.538"
          y2="16.821"
          gradientUnits="userSpaceOnUse">
          <stop stopColor="#FBCB9F" />
          <stop offset="1" stopColor="#7600D3" stopOpacity=".7" />
        </linearGradient>
      </defs>
    </svg>
  )
}
