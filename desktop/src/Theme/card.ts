import { cardAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(cardAnatomy.keys)
export const Card = defineMultiStyleConfig({
  baseStyle: definePartsStyle({
    container: {
      backgroundColor: "gray.50",
      borderColor: "gray.200",
      _dark: {
        borderColor: "gray.700",
        backgroundColor: "gray.900",
      },
    },
  }),
})
