import { tabsAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(tabsAnatomy.keys)

const muted = definePartsStyle((props) => {
  return {
    tab: {
      bg: mode("white", "black")(props),
      fontWeight: "medium",
      _selected: {
        bg: mode("gray.100", "gray.900")(props),
      },
      _hover: {
        bg: mode("gray.50", "gray.700")(props),
      },
    },
    tablist: {
      width: "fit-content",
      borderWidth: "thin",
      borderColor: "inherit",
      borderRadius: "md",
      "> *:not(:last-child, :first-child)": {
        borderRightWidth: "thin",
        borderRightColor: "inherit",
        borderRadius: "0",
      },
      "> :first-child": {
        borderTopLeftRadius: "md",
        borderBottomLeftRadius: "md",
        borderRightWidth: "thin",
        borderRightColor: "inherit",
      },
      "> :last-child": {
        borderTopRightRadius: "md",
        borderBottomRightRadius: "md",
      },
    },
  }
})

const mutedPopover = definePartsStyle((props) => {
  return {
    tab: {
      bg: mode("white", "gray.900")(props),
      fontWeight: "medium",
      _selected: {
        bg: mode("gray.100", "gray.800")(props),
      },
      _hover: {
        bg: mode("gray.50", "gray.700")(props),
      },
    },
    tablist: {
      width: "fit-content",
      borderWidth: "thin",
      borderColor: "inherit",
      borderRadius: "md",
      "> *:not(:last-child, :first-child)": {
        borderRightWidth: "thin",
        borderRightColor: "inherit",
        borderRadius: "0",
      },
      "> :first-child": {
        borderTopLeftRadius: "md",
        borderBottomLeftRadius: "md",
        borderRightWidth: "thin",
        borderRightColor: "inherit",
      },
      "> :last-child": {
        borderTopRightRadius: "md",
        borderBottomRightRadius: "md",
      },
    },
  }
})

const variants = {
  muted,
  "muted-popover": mutedPopover,
}

export const Tabs = defineMultiStyleConfig({ variants })
