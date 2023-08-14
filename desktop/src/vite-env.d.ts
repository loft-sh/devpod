/// <reference types="vite/client" />
declare const process: {
  env: {
    DEVPOD_PRO: boolean
    TAURI_DEBUG: boolean
    TAURI_PLATFORM: string
  }
}
