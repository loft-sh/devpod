import { ThemeProvider } from "./Theme"
import { invoke } from "@tauri-apps/api"
import { StrictMode, useEffect } from "react"
import ReactDOM from "react-dom/client"
import { DevPodProvider, SettingsProvider } from "./contexts"
import { RouterProvider } from "react-router"
import { router } from "./routes"
import "xterm/css/xterm.css"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ReactQueryDevtools } from "@tanstack/react-query-devtools"
import dayjs from "dayjs"
import relativeTime from "dayjs/plugin/relativeTime"

dayjs.extend(relativeTime)

const queryClient = new QueryClient({
  logger: {
    log(...args) {
      console.log(args)
    },
    warn(...args) {
      console.warn(args)
    },
    error(...args) {
      const maybeError = args[0]
      if (maybeError instanceof Error) {
        console.error(maybeError.name, maybeError.message, maybeError.cause, maybeError)

        return
      }

      console.error(args)
    },
  },
})

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Root />)

function Root() {
  // notifies underlying layer that ui is ready for communication
  useEffect(() => {
    invoke("ui_ready")
  }, [])

  return (
    <StrictMode>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <SettingsProvider>
            <DevPodProvider>
              <RouterProvider router={router} />
            </DevPodProvider>
          </SettingsProvider>
          {/* Will be disabled in production automatically */}
          <ReactQueryDevtools position="top-right" />
        </QueryClientProvider>
      </ThemeProvider>
    </StrictMode>
  )
}
