import { Box, Card, Image, useColorModeValue } from "@chakra-ui/react"
import { AnimatePresence, motion } from "framer-motion"
import React, { useId } from "react"

type TExampleCardProps = {
  image?: string
  source?: string

  isSelected?: boolean
  imageNode?: React.ReactNode
  onClick?: () => void
}

export function ExampleCard({ image, isSelected, imageNode, onClick }: TExampleCardProps) {
  const hoverBackgroudColor = useColorModeValue("gray.50", "gray.800")
  const selectedProps = isSelected
    ? {
        boxShadow:
          "0px 0.6px 0.8px hsl(0deg 0% 0% / 0.09), -0.2px 2.5px 3.3px -1.3px hsl(0deg 0% 0% / 0.18)",
      }
    : {}

  return (
    <Card
      variant="unstyled"
      width="32"
      height="32"
      alignItems="center"
      display="flex"
      justifyContent="center"
      cursor="pointer"
      padding="2"
      boxSizing="border-box"
      position="relative"
      backgroundColor="transparent"
      _hover={{ backgroundColor: hoverBackgroudColor }}
      {...(onClick ? { onClick } : {})}
      {...selectedProps}>
      {imageNode ? (
        imageNode
      ) : (
        <Image
          objectFit="cover"
          overflow="hidden"
          maxWidth="28"
          padding="2"
          width="fill"
          height="fill"
          src={image}
        />
      )}
      <AnimatePresence>
        {isSelected && (
          <Box
            as={motion.div}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            position="absolute"
            width="full"
            height="3"
            bottom={"17px"}>
            <Box as={Glow} width="full" />
          </Box>
        )}
      </AnimatePresence>
    </Card>
  )
}

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
