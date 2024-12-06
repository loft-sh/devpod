import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ReactQueryDevtools } from "@tanstack/react-query-devtools"
import dayjs from "dayjs"
import relativeTime from "dayjs/plugin/relativeTime"
import { StrictMode } from "react"
import ReactDOM from "react-dom/client"
import { Location, RouterProvider } from "react-router"
import "@xterm/xterm/css/xterm.css"
import { ThemeProvider } from "./Theme"
import { SettingsProvider } from "./contexts"
import { router } from "./routes"

dayjs.extend(relativeTime)

const queryClient = new QueryClient()

// TODO: Clean up :)
let render = true
const l = localStorage.getItem("devpod-location-current")
if (l) {
  const loc = JSON.parse(l) as Location
  if (window.location.pathname !== loc.pathname) {
    window.location.pathname = loc.pathname
    render = false
  }
}

if (render) {
  ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Root />)
}

function Root() {
  return (
    <StrictMode>
      <SettingsProvider>
        <ThemeProvider>
          <QueryClientProvider client={queryClient}>
            <RouterProvider router={router} />
            {/* Will be disabled in production automatically */}
            <ReactQueryDevtools
              position="bottom-right"
              toggleButtonProps={{ style: { margin: "0.5em 0.5em 2rem" } }}
            />
          </QueryClientProvider>
        </ThemeProvider>
      </SettingsProvider>
    </StrictMode>
  )
}
