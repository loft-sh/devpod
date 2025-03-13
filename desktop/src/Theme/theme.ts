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
import { Text } from "./text"
import { Textarea } from "./textarea"

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
    variant: "primary",
  },
  variants: {
    muted(props) {
      return { color: mode("gray.600", "gray.400")(props) }
    },
    primary(props) {
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

const TooltipTheme = defineStyleConfig({
  baseStyle(props) {
    return {
      bg: mode("gray.800", "gray.200")(props),
      color: mode("gray.100", "gray.800")(props),
    }
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
      50: "#F8EFFF",
      100: "#F0DFFF",
      200: "#D8ABFF",
      300: "#BF76FF",
      400: "#B157FF",
      500: "#A640FF",
      600: "#9B29FF",
      700: "#8600DC",
      800: "#7100B9",
      900: "#40006A",
    },
    gray: {
      50: "#F7F5F9",
      100: "#ECE8F0",
      200: "#DCD6E1",
      300: "#C5BFC9",
      400: "#ABA5B0",
      500: "#948E99",
      600: "#7C7581",
      700: "#655E69",
      800: "#4A464D",
      900: "#2C2630",
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
    Text,
    Textarea,
    Tooltip: TooltipTheme,
  },
} satisfies ThemeOverride) as Theme
