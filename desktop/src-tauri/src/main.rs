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
mod window;
mod workspaces;

use custom_protocol::{CustomProtocol, OpenWorkspaceMsg};
use log::error;
use serde::Serialize;
use std::{
    collections::VecDeque,
    sync::{Arc, Mutex},
};
use system_tray::SystemTray;
use tauri::{Manager, Menu, Wry};
use tokio::sync::mpsc::{self, Sender};
use workspaces::WorkspacesState;

pub type AppHandle = tauri::AppHandle<Wry>;

#[derive(Debug)]
pub struct AppState {
    workspaces: Arc<Mutex<WorkspacesState>>,
    ui_messages: Sender<UiMessage>,
}

#[derive(Debug, Serialize, Clone)]
enum UiMessage {
    Ready,
    ExitRequested,
    ShowDashboard,
    OpenWorkspace(OpenWorkspaceMsg),
}

fn main() -> anyhow::Result<()> {
    fix_path_env::fix()?;
    let ctx = tauri::generate_context!();
    let app_name = ctx.package_info().name.to_string();
    let menu = Menu::os_default(&app_name);

    let custom_protocol = CustomProtocol::init();

    let system_tray = SystemTray::new();
    let system_tray_event_handler = system_tray.get_event_handler();

    let (tx, mut rx) = mpsc::channel::<UiMessage>(10);

    tauri::Builder::default()
        .manage(AppState {
            workspaces: Arc::new(Mutex::new(WorkspacesState::default())),
            ui_messages: tx.clone(),
        })
        .plugin(logging::build_plugin())
        .plugin(tauri_plugin_store::Builder::default().build())
        .system_tray(system_tray.build())
        .menu(menu)
        .setup(move |app| {
            let window = app.get_window("main").unwrap();
            window::setup(&window);

            workspaces::setup(&app.handle(), app.state());
            action_logs::setup(&app.handle())?;
            custom_protocol.setup(app.handle());

            let app_handle = app.handle();
            tauri::async_runtime::spawn(async move {
                let mut is_ready = false;
                let mut messages: VecDeque<UiMessage> = VecDeque::new();

                while let Some(ui_msg) = rx.recv().await {
                    match ui_msg {
                        UiMessage::Ready => {
                            is_ready = true;

                            app_handle.get_window("main").and_then(|w| Some(w.show()));
                            while let Some(msg) = messages.pop_front() {
                                let _ = app_handle.emit_all("event", msg);
                            }
                        }
                        UiMessage::ExitRequested => {
                            is_ready = false;
                        }
                        UiMessage::OpenWorkspace(..) => {
                            if is_ready {
                                app_handle.get_window("main").and_then(|w| Some(w.show()));
                                let _ = app_handle.emit_all("event", ui_msg);
                            } else {
                                // recreate window
                                let _ = window::new_main(&app_handle, app_name.to_string());
                                messages.push_back(ui_msg);
                            }
                        }
                        UiMessage::ShowDashboard => {
                            if is_ready {
                                app_handle.get_window("main").and_then(|w| Some(w.show()));
                                let _ = app_handle.emit_all("event", ui_msg);
                            } else {
                                let _ = window::new_main(&app_handle, app_name.to_string());
                                messages.push_back(ui_msg);
                            }
                        }
                    }
                }
            });

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
        .run(move |app, event| {
            let exit_requested_tx = tx.clone();

            match event {
                // Prevents app from exiting when last window is closed, leaving the system tray active
                tauri::RunEvent::ExitRequested { api, .. } => {
                    tauri::async_runtime::block_on(async move {
                        if let Err(err) = exit_requested_tx.send(UiMessage::ExitRequested).await {
                            error!("Failed to broadcast UI ready message: {:?}", err);
                        }
                    });
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
            }
        });

    Ok(())
}
