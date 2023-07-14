import { Theme, ThemeOverride, Tooltip, defineStyleConfig, extendTheme } from "@chakra-ui/react"
import { Menu } from "./menu"
import { Switch } from "./switch"

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
    announcement({ theme }) {
      const from = theme.colors.primary["900"]
      const to = theme.colors.primary["600"]

      return {
        color: "white",
        transition: "background 150ms",
        fontWeight: "regular",
        background: `linear-gradient(170deg, ${from} 15%, ${to})`,
        backgroundSize: "130% 130%",
        _hover: {
          backgroundPosition: "90% 50%",
        },
        _active: {
          boxShadow: "inset 0 0 3px 2px rgba(0, 0, 0, 0.2)",
        },
      }
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
        },
        td: {
          userSelect: "auto",
        },
        code: {
          userSelect: "auto",
        },
        "input::placeholder": {
          color: colorMode === "light" ? "gray.500" : "gray.400",
        },
      }
    },
  },
  colors: {
    primary: {
      400: "#CA60FF",
      500: "#BA50FF",
      600: "#AA40EE",
      800: "#8E00EB",
      900: "#40006A",
    },
  },
  config: {
    initialColorMode: "system",
    useSystemColorMode: true,
  },
  components: {
    Button,
    Code,
    Menu,
    Switch,
  },
} satisfies ThemeOverride) as Theme
