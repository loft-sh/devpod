import { Theme, ThemeOverride, Tooltip, defineStyleConfig, extendTheme } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"
import { Button } from "./button"
import { Card } from "./card"
import { Checkbox } from "./checkbox"
import { Form } from "./form"
import { Input } from "./input"
import { Menu } from "./menu"
import { Modal } from "./modal"
import { Popover } from "./popover"
import { Radio } from "./radio"
import { Select } from "./select"
import { Switch } from "./switch"
import { Tabs } from "./tabs"
import { Tag } from "./tag"

const Code = defineStyleConfig({
  variants: {
    decorative(props) {
      return {
        backgroundColor: "primary.400",
        color: mode("white", "gray.900")(props),
      }
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

      return { color: mode(primary["800"], primary["400"])(props) }
    },
  },
})

const FormError = defineStyleConfig({
  baseStyle: {
    text: {
      userSelect: "text",
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
          position: "fixed",
        },
        body: {
          background: "transparent",
          userSelect: "none",
          cursor: "default",
        },
        td: {
          userSelect: "text",
        },
        code: {
          userSelect: "text",
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
    text: {
      secondary: "#465E75",
      tertiary: "#5C7997",
    },
    divider: {
      main: "#B0C3D6",
      light: "#DCE5EE",
      dark: "#465E75",
    },
    background: {
      darkest: "rgb(16, 18, 20)",
    },
  },
  config: {
    initialColorMode: "system",
    useSystemColorMode: true,
  },
  components: {
    Button,
    Card,
    Code,
    Menu,
    Switch,
    Tabs,
    Checkbox,
    Radio,
    Link,
    Form,
    FormError,
    Popover,
    Modal,
    Tag,
    Input,
    Select,
  },
} satisfies ThemeOverride) as Theme
