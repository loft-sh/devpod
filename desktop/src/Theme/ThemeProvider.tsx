import { ChakraProvider, ColorModeScript, ToastProviderProps } from "@chakra-ui/react"
import { ReactNode } from "react"
import { theme } from "./theme"
const toastOptions: ToastProviderProps = { defaultOptions: { variant: "subtle" } }

export function ThemeProvider({ children }: Readonly<{ children?: ReactNode }>) {
  return (
    <>
      <ColorModeScript initialColorMode={theme.config.initialColorMode} />
      <ChakraProvider theme={theme} toastOptions={toastOptions}>
        {children}
      </ChakraProvider>
    </>
  )
}
