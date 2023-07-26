import { ChakraProvider, Theme, ToastProviderProps, extendTheme } from "@chakra-ui/react"
import { ReactNode, useEffect, useState } from "react"
import { TSettings, useSettings } from "../contexts"
import { theme as initialTheme } from "./theme"
const toastOptions: ToastProviderProps = { defaultOptions: { variant: "subtle" } }

const fontSize: Record<TSettings["zoom"], string> = {
  sm: "12px",
  md: "14px",
  lg: "16px",
  xl: "18px",
}

export function ThemeProvider({ children }: Readonly<{ children?: ReactNode }>) {
  const settings = useSettings()
  const [theme, setTheme] = useState<Theme>(initialTheme)

  useEffect(() => {
    setTheme(
      (current) =>
        extendTheme(
          {
            styles: {
              global: {
                html: {
                  fontSize: fontSize[settings.zoom],
                },
              },
            },
          },
          current
        ) as Theme
    )
  }, [settings])

  return (
    <ChakraProvider theme={theme} toastOptions={toastOptions}>
      {children}
    </ChakraProvider>
  )
}
