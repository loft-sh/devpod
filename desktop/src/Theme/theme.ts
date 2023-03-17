// We need the `.cts` filename for chakra-cli to properly generate our types
import { extendTheme, Theme, ThemeOverride } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"

export const theme = extendTheme({
  styles: {
    global({ colorMode }) {
      return {
        html: {
          fontSize: "14px",
          overflow: "hidden",
        },
        body: {
          backgroundColor: mode("gray.50", "gray.900")({ colorMode }),
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
  colors: { primary: "#BA50FF" },
  config: {
    initialColorMode: "system",
    useSystemColorMode: true,
  },
} satisfies ThemeOverride) as Theme
