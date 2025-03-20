import { menuAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(menuAnatomy.keys)
export const Menu = defineMultiStyleConfig({
  baseStyle: definePartsStyle((props) => {
    return {
      item: {
        fontSize: "sm",
        bg: mode("white", "gray.900")(props),
        _selected: {
          bg: mode("gray.200", "gray.800")(props),
        },
        _hover: {
          bg: mode("gray.100", "gray.700")(props),
        },
        borderColor: "gray.200",
        _dark: {
          borderColor: "gray.700",
        },
      },
      list: {
        bg: mode("white", "gray.900")(props),
        borderColor: "gray.200",
        _dark: {
          borderColor: "gray.700",
        },
      },
      divider: {
        borderColor: "gray.200",
        _dark: {
          borderColor: "gray.700",
        },
      },
    }
  }),
})
