import { popoverAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(
  popoverAnatomy.keys
)
export const Popover = defineMultiStyleConfig({
  baseStyle: definePartsStyle(({ theme }) => {
    return {
      content: {
        boxShadow: theme.shadows.xl,
        _focusVisible: {
          outline: "2px solid transparent",
          outlineOffset: "2px",
          boxShadow: theme.shadows.xl,
        },
      },
      popper: {
        zIndex: theme.zIndices.popover,
      },
      header: {
        display: "flex",
        alignItems: "center",
        flexFlow: "row nowrap",
        padding: 4,
        spacing: 0,
        justifyContent: "space-between",
        borderBottomWidth: "thin",
        borderColor: "inherit",
        fontWeight: "bold",
        p: {
          fontWeight: "normal",
        },
      },
    }
  }),
})
