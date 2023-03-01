import { ThemeProvider } from "./ThemeProvider"
import { invoke } from "@tauri-apps/api"
import { StrictMode, useEffect } from "react"
import ReactDOM from "react-dom/client"
import { App } from "./App"
import { DevpodProvider } from "./DevpodContext"

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Root />)

function Root() {
  // notifies underlying layer that ui is ready for communication
  useEffect(() => {
    invoke("ui_ready")
  }, [])

  return (
    <StrictMode>
      <ThemeProvider>
        <DevpodProvider>
          <App />
        </DevpodProvider>
      </ThemeProvider>
    </StrictMode>
  )
}
