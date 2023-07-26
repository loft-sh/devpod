#![cfg_attr(
    all(not(debug_assertions), target_os = "windows"),
    windows_subsystem = "windows"
)]

#[cfg(target_os = "macos")]
#[macro_use]
extern crate objc;

mod action_logs;
mod commands;
mod community_contributions;
mod custom_protocol;
mod install_cli;
mod logging;
mod providers;
mod system_tray;
mod ui_ready;
mod util;
mod window;
mod workspaces;

use community_contributions::CommunityContributions;
use custom_protocol::{CustomProtocol, OpenWorkspaceMsg};
use log::{error, info, warn};
use serde::Serialize;
use std::{
    collections::VecDeque,
    sync::{Arc, Mutex},
};
use system_tray::SystemTray;
use tauri::{Manager, Menu, Wry};
use tokio::{
    sync::mpsc::{self, Sender},
    time::sleep,
};
use workspaces::WorkspacesState;

pub type AppHandle = tauri::AppHandle<Wry>;

#[derive(Debug)]
pub struct AppState {
    workspaces: Arc<Mutex<WorkspacesState>>,
    community_contributions: Arc<Mutex<CommunityContributions>>,
    ui_messages: Sender<UiMessage>,
}

#[derive(Debug, Serialize, Clone)]
#[serde(tag = "type")]
enum UiMessage {
    Ready,
    ExitRequested,
    ShowDashboard,
    ShowToast(ShowToastMsg),
    OpenWorkspace(OpenWorkspaceMsg),
    OpenWorkspaceFailed(custom_protocol::ParseError),
}

#[derive(Debug, Serialize, Clone)]
struct ShowToastMsg {
    title: String,
    message: String,
    status: ToastStatus,
}

// WARN: Needs to match the UI's toast status
#[derive(Debug, Serialize, Clone)]
#[serde(rename_all = "lowercase")]
enum ToastStatus {
    Success,
    Error,
    Warning,
    Info,
    Loading,
}

fn main() -> anyhow::Result<()> {
    fix_path_env::fix()?;
    let ctx = tauri::generate_context!();
    let app_name = ctx.package_info().name.to_string();
    let menu = if cfg!(target_os = "macos") {
        Menu::os_default(&app_name)
    } else {
        Menu::new()
    };

    let custom_protocol = CustomProtocol::init();
    let contributions = community_contributions::init()?;

    let system_tray = SystemTray::new();
    let system_tray_event_handler = system_tray.get_event_handler();

    let (tx, mut rx) = mpsc::channel::<UiMessage>(10);

    let app = tauri::Builder::default()
        .manage(AppState {
            workspaces: Arc::new(Mutex::new(WorkspacesState::default())),
            community_contributions: Arc::new(Mutex::new(contributions)),
            ui_messages: tx.clone(),
        })
        .plugin(logging::build_plugin())
        .plugin(tauri_plugin_store::Builder::default().build())
        .system_tray(system_tray.build_tray(vec![Box::new(&WorkspacesState::default())]))
        .menu(menu)
        .setup(move |app| {
            info!("Setup application");

            providers::check_dangling_provider(&app.handle());
            let window_helper = window::WindowHelper::new(&app.handle());

            let window = app.get_window("main").unwrap();
            window_helper.setup(&window);

            workspaces::setup(&app.handle(), app.state());
            community_contributions::setup(app.state());
            action_logs::setup(&app.handle())?;
            custom_protocol.setup(app.handle());

            #[cfg(feature = "updater")]
            {
                let app_handle = app.handle();
                check_update(app_handle);
            }

            let app_handle = app.handle();
            tauri::async_runtime::spawn(async move {
                let mut is_ready = false;
                let mut messages: VecDeque<UiMessage> = VecDeque::new();

                while let Some(ui_msg) = rx.recv().await {
                    match ui_msg {
                        UiMessage::Ready => {
                            is_ready = true;

                            app_handle.get_window("main").map(|w| w.show());
                            while let Some(msg) = messages.pop_front() {
                                let emit_result = app_handle.emit_all("event", msg);
                                if let Err(err) = emit_result {
                                    warn!("Error sending message: {}", err);
                                }
                            }
                        }
                        UiMessage::ExitRequested => {
                            is_ready = false;
                        }
                        UiMessage::OpenWorkspace(..) => {
                            if is_ready {
                                app_handle.get_window("main").map(|w| w.show());
                                let _ = app_handle.emit_all("event", ui_msg);
                            } else {
                                // recreate window
                                let _ = window_helper.new_main(app_name.to_string());
                                messages.push_back(ui_msg);
                            }
                        }
                        UiMessage::OpenWorkspaceFailed(..) => {
                            if is_ready {
                                app_handle.get_window("main").map(|w| w.show());
                                let _ = app_handle.emit_all("event", ui_msg);
                            } else {
                                // recreate window
                                let _ = window_helper.new_main(app_name.to_string());
                                messages.push_back(ui_msg);
                            }
                        }
                        UiMessage::ShowDashboard => {
                            if is_ready {
                                app_handle.get_window("main").map(|w| w.show());
                                let _ = app_handle.emit_all("event", ui_msg);
                            } else {
                                // recreate window
                                let _ = window_helper.new_main(app_name.to_string());
                                messages.push_back(ui_msg);
                            }
                        }
                        UiMessage::ShowToast(..) => {
                            if is_ready {
                                app_handle.get_window("main").map(|w| w.show());
                                let _ = app_handle.emit_all("event", ui_msg);
                            } else {
                                // recreate window
                                let _ = window_helper.new_main(app_name.to_string());
                                messages.push_back(ui_msg);
                            }
                        }
                    }
                }
            });

            info!("Setup done");
            Ok(())
        })
        .on_system_tray_event(system_tray_event_handler)
        .invoke_handler(tauri::generate_handler![
            ui_ready::ui_ready,
            action_logs::write_action_log,
            action_logs::get_action_logs,
            action_logs::sync_action_logs,
            install_cli::install_cli,
            community_contributions::get_contributions,
        ])
        .build(ctx)
        .expect("error while building tauri application");

    info!("Run");

    app.run(move |app_handle, event| {
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
            tauri::RunEvent::WindowEvent { event, label, .. } => {
                if let tauri::WindowEvent::Destroyed = event {
                    providers::check_dangling_provider(app_handle);
                    #[cfg(target_os = "macos")]
                    {
                        let window_helper = window::WindowHelper::new(app_handle);
                        let window_count = app_handle.windows().len();
                        info!("Window \"{}\" destroyed, {} remaining", label, window_count);
                        if window_count == 0 {
                            window_helper.set_dock_icon_visibility(false);
                        }
                    }
                }
            }
            tauri::RunEvent::Exit => {
                providers::check_dangling_provider(app_handle);
            }
            _ => {}
        }
    });

    Ok(())
}

#[cfg(feature = "updater")]
fn check_update(app_handle: AppHandle) {
    tauri::async_runtime::spawn(async move {
        loop {
            match tauri::updater::builder(app_handle.clone()).check().await {
                Ok(update) => {
                    if update.is_update_available() {
                        // TODO: Might  need to be run on the main thread, check once repo is public
                        if let Err(e) = update.download_and_install().await {
                            error!("Failed to download and install update: {}", e)
                        }
                    }
                }
                Err(e) => {
                    error!("Failed to get update: {}", e);
                }
            }

            tokio::time::sleep(std::time::Duration::from_secs(60 * 10)).await;
        }
    });
}
