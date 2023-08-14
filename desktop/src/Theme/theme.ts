import { Theme, ThemeOverride, Tooltip, defineStyleConfig, extendTheme } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"
import { Menu } from "./menu"
import { Switch } from "./switch"
import { Tabs } from "./tabs"
import { Checkbox } from "./checkbox"
import { Radio } from "./radio"
import { Popover } from "./popover"
import { Button } from "./button"
import { Tag } from "./tag"

const Code = defineStyleConfig({
  variants: {
    decorative: {
      backgroundColor: "primary.400",
      color: "white",
    },
  },
})

const Link = defineStyleConfig({
  defaultProps: {
    variant: "muted",
  },
  variants: {
    muted(props) {
      const primary = props.theme.colors.primary

      return { color: mode(primary["900"], primary["200"])(props) }
    },
  },
})

const FormError = defineStyleConfig({
  baseStyle: {
    text: {
      userSelect: "auto",
      cursor: "text",
    },
  },
})

// It's ugly but it works: https://github.com/chakra-ui/chakra-ui/issues/1424#issuecomment-743342944
// Unfortunately there is no other way of overring the default placement.
Tooltip.defaultProps = { ...Tooltip.defaultProps, placement: "top" }

export const theme = extendTheme({
  styles: {
    global({ colorMode }) {
      return {
        html: {
          fontSize: "14px",
          overflow: "hidden",
          background: "transparent",
        },
        body: {
          background: "transparent",
          userSelect: "none",
          cursor: "default",
        },
        td: {
          userSelect: "auto",
        },
        code: {
          userSelect: "auto",
          cursor: "text",
        },
        "input::placeholder": {
          color: colorMode === "light" ? "gray.500" : "gray.400",
        },
      }
    },
  },
  colors: {
    primary: {
      200: "#E4ADFF",
      400: "#CA60FF",
      500: "#BA50FF",
      600: "#AA40EE",
      800: "#8E00EB",
      900: "#40006A",
    },
    background: {
      darkest: "rgb(16, 18, 20)",
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
    Switch,
    Tabs,
    Checkbox,
    Radio,
    Link,
    FormError,
    Popover,
    Tag,
  },
} satisfies ThemeOverride) as Theme
