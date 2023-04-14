import { defineStyleConfig, extendTheme, Theme, ThemeOverride } from "@chakra-ui/react"

const Button = defineStyleConfig({
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
  },
} satisfies ThemeOverride) as Theme
