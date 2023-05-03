import {
  createMultiStyleConfigHelpers,
  defineStyleConfig,
  extendTheme,
  Theme,
  Tooltip,
  ThemeOverride,
} from "@chakra-ui/react"
import { menuAnatomy } from "@chakra-ui/anatomy"

const Button = defineStyleConfig({
  defaultProps: {
    size: "sm",
  },
  variants: {
    primary: {
      color: "white",
      borderColor: "primary.600",
      borderWidth: 1,
      backgroundColor: "primary.500",
      _hover: {
        backgroundColor: "primary.600",
        _disabled: {
          background: "primary.500",
        },
      },
    },
  },
})

const Code = defineStyleConfig({
  variants: {
    decorative: {
      backgroundColor: "primary.400",
      color: "white",
    },
  },
})
const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(menuAnatomy.keys)
const Menu = defineMultiStyleConfig({
  baseStyle: definePartsStyle({
    item: {
      fontSize: "sm",
    },
  }),
})

// It's ugly but it works: https://github.com/chakra-ui/chakra-ui/issues/1424#issuecomment-743342944
// Unfortunately there is no other way of overring the default placement.
Tooltip.defaultProps = { ...Tooltip.defaultProps, placement: "top" }

export const theme = extendTheme({
  styles: {
    global() {
      return {
        html: {
          fontSize: "14px",
          overflow: "hidden",
          backgroun: "transparent",
        },
        body: {
          background: "transparent",
          userSelect: "none",
        },
        td: {
          userSelect: "auto",
        },
        code: {
          userSelect: "auto",
        },
      }
    },
  },
  colors: {
    primary: {
      400: "#CA60FF",
      500: "#BA50FF",
      600: "#AA40EE",
    },
  },
  config: {
    initialColorMode: "light",
    useSystemColorMode: false,
  },
  components: {
    Button,
    Code,
    Menu,
  },
} satisfies ThemeOverride) as Theme
