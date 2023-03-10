import { ThemeProvider } from "./ThemeProvider"
import { invoke } from "@tauri-apps/api"
import { StrictMode, useEffect } from "react"
import ReactDOM from "react-dom/client"
import { App } from "./App"
import { DevPodProvider } from "./contexts/DevPodContext/DevPodContext"
import "xterm/css/xterm.css"

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Root />)

function Root() {
  // notifies underlying layer that ui is ready for communication
  useEffect(() => {
    invoke("ui_ready")
  }, [])

  return (
    <StrictMode>
      <ThemeProvider>
        <DevPodProvider>
          <App />
        </DevPodProvider>
      </ThemeProvider>
    </StrictMode>
  )
}
