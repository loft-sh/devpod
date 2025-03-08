import { popoverAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers, cssVar } from "@chakra-ui/react"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(
  popoverAnatomy.keys
)
export const Popover = defineMultiStyleConfig({
  baseStyle: definePartsStyle((props) => {
    const theme = props.theme
    let bg = theme.colors.white
    if (props.colorMode == "dark") {
      bg = theme.colors.gray["900"]
    }

    return {
      content: {
        borderColor: "gray.200",
        bg,
        boxShadow: theme.shadows.xl,
        _focusVisible: {
          outline: "2px solid transparent",
          outlineOffset: "2px",
          boxShadow: theme.shadows.xl,
        },
        [cssVar("popper-bg").variable]: bg,
        _dark: {
          [cssVar("popper-arrow-bg").variable]: bg,
          borderColor: "gray.700",
        },
      },
      arrow: { bg },
      popper: {
        zIndex: theme.zIndices.popover,
        bg,
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
