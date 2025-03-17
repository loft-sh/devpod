import { Logger, QueryClient, QueryClientProvider } from "@tanstack/react-query"
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
import { client } from "./client"
import { ColorModeScript } from "@chakra-ui/react"

dayjs.extend(relativeTime)

const logger: Logger | undefined = import.meta.env.PROD
  ? {
      log: () => {
        // noop in prod
      },
      warn: (...args: any[]) => {
        client.log("warn", args.join(" "))
      },
      error: (...args: any[]) => {
        client.log("error", args.join(" "))
      },
    }
  : undefined
const queryClient = new QueryClient({ logger })

let render = true
const l = localStorage.getItem("devpod-location-current") // check usePreserveLocation before changing this
if (l) {
  const loc = JSON.parse(l) as Location
  if (window.location.pathname !== loc.pathname) {
    window.location.href = loc.pathname + loc.search
    render = false
  }
}

if (render) {
  ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Root />)
}
// force chakra to determine color mode on startup
localStorage.removeItem("chakra-ui-color-mode")

function Root() {
  return (
    <StrictMode>
      <SettingsProvider>
        <ColorModeScript initialColorMode={"system"} />
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
