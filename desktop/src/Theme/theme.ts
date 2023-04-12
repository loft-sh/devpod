import { defineStyleConfig, extendTheme, Theme, ThemeOverride } from "@chakra-ui/react"

const Button = defineStyleConfig({
  variants: {
    primary: {
      color: "white",
      borderColor: "primary.400",
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
      400: "#AA40EE",
      500: "#BA50FF",
      600: "#CA60FF",
    },
  },
  config: {
    initialColorMode: "light",
    useSystemColorMode: false,
  },
  components: {
    Button,
  },
} satisfies ThemeOverride) as Theme
