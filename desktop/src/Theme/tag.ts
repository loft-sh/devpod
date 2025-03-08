import { tagAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(tagAnatomy.keys)

export const Tag = defineMultiStyleConfig({
  baseStyle: definePartsStyle((props) => {
    return {
      container: {
        bg: mode("gray.200", "gray.700")(props),
        color: mode("gray.800", "gray.100")(props),
      },
    }
  }),
  variants: {
    ghost: definePartsStyle(() => {
      return {
        container: {
          bg: "transparent",
        },
      }
    }),
  },
})
