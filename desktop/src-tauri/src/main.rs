#![cfg_attr(
    all(not(debug_assertions), target_os = "windows"),
    windows_subsystem = "windows"
)]

mod commands;
mod logging;
mod system_tray;
mod ui_ready;
mod new_workspace_id;
mod providers;
mod util;
mod workspaces;

use providers::ProvidersState;
use std::sync::{Arc, Mutex};
use system_tray::SystemTray;
use tauri::Manager;
use tauri_plugin_deep_link;
use workspaces::WorkspacesState;
use ui_ready::ui_ready;
use new_workspace_id::new_workspace_id;

// Should match the one from `tauri.config.json"
const APP_IDENTIFIER: &str = "sh.loft.devpod-desktop";

#[derive(Debug)]
pub struct AppState {
    providers: Arc<Mutex<ProvidersState>>,
    workspaces: Arc<Mutex<WorkspacesState>>,
}

fn main() {
    tauri_plugin_deep_link::prepare(APP_IDENTIFIER);

    let system_tray = SystemTray::new();
    let system_tray_event_handler = system_tray.get_event_handler();

    tauri::Builder::default()
        .manage(AppState {
            providers: Arc::new(Mutex::new(ProvidersState::default())),
            workspaces: Arc::new(Mutex::new(WorkspacesState::default())),
        })
        .system_tray(system_tray.build())
        .plugin(logging::build_plugin())
        .setup(|app| {
            let handler = |url| {
                println!("App opened with URL: {:?}", url);
            };

            tauri_plugin_deep_link::register("test-scheme", handler)
                .expect("should be able to listen to custom protocols");

            #[cfg(debug_assertions)] // open browser devtools automatically during development
            {
                let window = app.get_window("main").unwrap();
                window.open_devtools();
            }

            Ok(())
        })
        .on_system_tray_event(system_tray_event_handler)
        .invoke_handler(tauri::generate_handler![ui_ready, new_workspace_id])
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|_app, event| match event {
            // Prevents app from exiting when last window is closed, leaving the system tray active
            tauri::RunEvent::ExitRequested { api, .. } => {
                api.prevent_exit();
            }
            _ => {}
        });
}
