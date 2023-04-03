#![cfg_attr(
    all(not(debug_assertions), target_os = "windows"),
    windows_subsystem = "windows"
)]

mod action_logs;
mod commands;
mod custom_protocol;
mod logging;
mod providers;
mod system_tray;
mod ui_ready;
mod util;
mod workspaces;

use custom_protocol::{CustomProtocol, OpenWorkspaceMsg};
use std::sync::{Arc, Mutex};
use system_tray::SystemTray;
use tauri::{Manager, Menu};
use workspaces::WorkspacesState;

#[derive(Debug)]
pub struct AppState {
    workspaces: Arc<Mutex<WorkspacesState>>,
    launch_msg: Mutex<Option<OpenWorkspaceMsg>>,
}

fn main() {
    let ctx = tauri::generate_context!();
    let app_name = &ctx.package_info().name;
    let menu = Menu::os_default(app_name);

    let custom_protocol = CustomProtocol::init();

    let system_tray = SystemTray::new();
    let system_tray_event_handler = system_tray.get_event_handler();

    tauri::Builder::default()
        .manage(AppState {
            workspaces: Arc::new(Mutex::new(WorkspacesState::default())),
            launch_msg: Mutex::new(None),
        })
        .plugin(logging::build_plugin())
        .plugin(tauri_plugin_store::Builder::default().build())
        .system_tray(system_tray.build())
        .menu(menu)
        .setup(move |app| {
            action_logs::setup(&app.handle())?;

            custom_protocol.setup(app.handle());

            let window = app.get_window("main").unwrap();
            setup_window(&window);

            Ok(())
        })
        .on_system_tray_event(system_tray_event_handler)
        .invoke_handler(tauri::generate_handler![
            ui_ready::ui_ready,
            action_logs::write_action_log,
            action_logs::get_action_logs
        ])
        .build(ctx)
        .expect("error while building tauri application")
        .run(|app, event| match event {
            // Prevents app from exiting when last window is closed, leaving the system tray active
            tauri::RunEvent::ExitRequested { api, .. } => {
                api.prevent_exit();
            }
            tauri::RunEvent::WindowEvent { event, .. } => {
                if let tauri::WindowEvent::Destroyed = event {
                    providers::check_dangling_provider(app);
                }
            }
            tauri::RunEvent::Exit => {
                providers::check_dangling_provider(app);
            }
            _ => {}
        });
}

fn setup_window(window: &tauri::Window<tauri::Wry>) {
    // open browser devtools automatically during development
    #[cfg(debug_assertions)]
    {
        window.open_devtools();
    }

    // Window vibrancy
    #[cfg(target_os = "macos")]
    {
        window_vibrancy::apply_vibrancy(
            window,
            window_vibrancy::NSVisualEffectMaterial::HudWindow,
            None,
            None,
        )
        .expect("Unsupported platform! 'apply_vibrancy' is only supported on macOS");
    }
    #[cfg(target_os = "windows")]
    {
        window_vibrancy::apply_blur(window, Some((18, 18, 18, 125)))
            .expect("Unsupported platform! 'apply_blur' is only supported on Windows");
    }
}
