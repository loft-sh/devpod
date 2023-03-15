import { ThemeProvider } from "./ThemeProvider"
import { invoke } from "@tauri-apps/api"
import { StrictMode, useEffect } from "react"
import ReactDOM from "react-dom/client"
import { DevPodProvider } from "./contexts"
import { RouterProvider } from "react-router"
import { router } from "./routes"
import "xterm/css/xterm.css"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ReactQueryDevtools } from "@tanstack/react-query-devtools"

const queryClient = new QueryClient()

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
          <DevPodProvider>
            <RouterProvider router={router} />
          </DevPodProvider>
          {/* Will be disabled in production automatically */}
          <ReactQueryDevtools position="top-right" />
        </QueryClientProvider>
      </ThemeProvider>
    </StrictMode>
  )
}
