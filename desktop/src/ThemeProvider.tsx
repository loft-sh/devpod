import { ChakraProvider, extendTheme } from "@chakra-ui/react"
import { ReactNode } from "react"

const theme = extendTheme({
  styles: {
    global: {
      html: {
        fontSize: "14px",
        overflow: "hidden",
      },
      body: {
        userSelect: "none",
      },
      td: {
        userSelect: "auto",
      },
      code: {
        userSelect: "auto",
      },
    },
  },
  colors: { primary: "#BA50FF" },
})

export function ThemeProvider({ children }: Readonly<{ children?: ReactNode }>) {
  return <ChakraProvider theme={theme}>{children}</ChakraProvider>
}
