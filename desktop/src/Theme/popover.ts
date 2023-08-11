import { popoverAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(
  popoverAnatomy.keys
)
export const Popover = defineMultiStyleConfig({
  baseStyle: definePartsStyle(({ theme }) => {
    return {
      content: {
        backgroundColor: "red",
        boxShadow: theme.shadows.xl,
        _focusVisible: {
          outline: "2px solid transparent",
          outlineOffset: "2px",
          boxShadow: theme.shadows.xl,
        },
      },
    }
  }),
})
