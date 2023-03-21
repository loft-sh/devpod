import { ChakraProvider, ColorModeScript } from "@chakra-ui/react"
import { ReactNode } from "react"
import { theme } from "./theme"

export function ThemeProvider({ children }: Readonly<{ children?: ReactNode }>) {
  return (
    <>
      <ColorModeScript initialColorMode={theme.config.initialColorMode} />
      <ChakraProvider theme={theme}>{children}</ChakraProvider>
    </>
  )
}
