/// <reference types="vite/client" />
declare const process: {
  env: {
    TAURI_DEBUG: boolean
    TAURI_PLATFORM: string
  }
}
