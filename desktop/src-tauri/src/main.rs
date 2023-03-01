#![cfg_attr(
    all(not(debug_assertions), target_os = "windows"),
    windows_subsystem = "windows"
)]

mod commands;
mod providers;
mod system_tray;
mod workspaces;

use commands::DevpodCommandError;
use log::info;
use providers::ProvidersState;
use system_tray::SystemTray;
use tauri::{plugin::TauriPlugin, AppHandle, Manager, Wry};
use tauri_plugin_deep_link;
use tauri_plugin_log::LogTarget;
use workspaces::WorkspacesState;

// Should match the one from `tauri.config.json"
const APP_IDENTIFIER: &str = "sh.loft.devpod-desktop";

#[tauri::command]
fn ui_ready(app_handle: AppHandle) -> Result<(), DevpodCommandError> {
    let providers = ProvidersState::load()?;
    // TODO: store state in tauri?
    app_handle
        .emit_all("providers", &providers)
        .expect("should be able to emit providers");

    let workspaces = WorkspacesState::load()?;
    app_handle
        .emit_all("workspaces", &workspaces)
        .expect("should be able to emit workspaces");

    let new_menu =
        SystemTray::new().build_with_submenus(vec![Box::new(workspaces), Box::new(providers)]);
    app_handle
        .tray_handle()
        .set_menu(new_menu)
        .expect("should be able to set menu");

    Ok(())
}

fn main() {
    tauri_plugin_deep_link::prepare(APP_IDENTIFIER);

    let system_tray = SystemTray::new();
    let system_tray_event_handler = system_tray.get_event_handler();

    tauri::Builder::default()
        .system_tray(system_tray.build())
        .plugin(build_log_plugin())
        .setup(|app| {
            let handler = |url| {
                info!("hello world");
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
        .invoke_handler(tauri::generate_handler![ui_ready])
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|_app, event| match event {
            // Prevents app from exiting when last window is closed, leaving the system tray
            // active
            tauri::RunEvent::ExitRequested { api, .. } => {
                api.prevent_exit();
            }
            _ => {}
        });
}

fn build_log_plugin() -> TauriPlugin<Wry> {
    // TODO: ADJUST LEVELS
    tauri_plugin_log::Builder::default()
        .targets([
            #[cfg(debug_assertions)] // only enable during development
            LogTarget::Stdout,
            #[cfg(not(debug_assertions))] // only enable in release builds
            LogTarget::LogDir,
        ])
        .build()
}
