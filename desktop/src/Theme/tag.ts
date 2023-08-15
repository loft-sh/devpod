import { tagAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(tagAnatomy.keys)

export const Tag = defineMultiStyleConfig({
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
