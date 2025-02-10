/// <reference types="vite/client" />
declare const process: {
  env: {
    TAURI_ENV_DEBUG: boolean
    TAURI_ENV_PLATFORM: string
  }
}
