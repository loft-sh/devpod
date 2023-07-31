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
mod fix_env;
mod install_cli;
mod logging;
mod providers;
mod settings;
mod system_tray;
mod ui_messages;
mod ui_ready;
#[cfg(feature = "enable-updater")]
mod updates;
mod util;
mod window;
mod workspaces;

use community_contributions::CommunityContributions;
use custom_protocol::CustomProtocol;
use log::{error, info};
use std::sync::{Arc, Mutex};
use system_tray::SystemTray;
use tauri::{Manager, Menu, Wry};
use tokio::sync::mpsc::{self, Sender};
use ui_messages::UiMessage;
use workspaces::WorkspacesState;

pub type AppHandle = tauri::AppHandle<Wry>;

#[derive(Debug)]
pub struct AppState {
    workspaces: Arc<Mutex<WorkspacesState>>,
    community_contributions: Arc<Mutex<CommunityContributions>>,
    ui_messages: Sender<UiMessage>,
    #[cfg(feature = "enable-updater")]
    releases: Arc<Mutex<updates::Releases>>,
    #[cfg(feature = "enable-updater")]
    pending_update: Arc<Mutex<Option<updates::Release>>>,
    #[cfg(feature = "enable-updater")]
    update_installed: Arc<Mutex<bool>>,
}

fn main() -> anyhow::Result<()> {
    // https://unix.stackexchange.com/questions/82620/gui-apps-dont-inherit-path-from-parent-console-apps
    fix_env::fix_env("PATH")?;

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

    let (tx, rx) = mpsc::channel::<UiMessage>(10);

    let mut app_builder = tauri::Builder::default()
        .manage(AppState {
            workspaces: Arc::new(Mutex::new(WorkspacesState::default())),
            community_contributions: Arc::new(Mutex::new(contributions)),
            ui_messages: tx.clone(),
            #[cfg(feature = "enable-updater")]
            releases: Arc::new(Mutex::new(updates::Releases::default())),
            #[cfg(feature = "enable-updater")]
            pending_update: Arc::new(Mutex::new(None)),
            #[cfg(feature = "enable-updater")]
            update_installed: Arc::new(Mutex::new(false)),
        })
        .plugin(logging::build_plugin())
        .plugin(tauri_plugin_store::Builder::default().build())
        .system_tray(system_tray.build_tray(vec![Box::new(&WorkspacesState::default())]))
        .menu(menu)
        .setup(move |app| {
            info!("Setup application");

            providers::check_dangling_provider(&app.handle());
            let window_helper = window::WindowHelper::new(app.handle());

            let window = app.get_window("main").unwrap();
            window_helper.setup(&window);

            workspaces::setup(&app.handle(), app.state());
            community_contributions::setup(app.state());
            action_logs::setup(&app.handle())?;
            custom_protocol.setup(app.handle());

            #[cfg(feature = "enable-updater")]
            let app_handle = app.handle();
            #[cfg(feature = "enable-updater")]
            tauri::async_runtime::spawn(async move {
                let update_helper = updates::UpdateHelper::new(&app_handle);
                if let Ok(releases) = update_helper.fetch_releases().await {
                    let state = app_handle.state::<AppState>();
                    let mut releases_state = state.releases.lock().unwrap();
                    *releases_state = releases;
                }

                update_helper.poll().await;
            });

            let app_handle = app.handle();
            tauri::async_runtime::spawn(async move {
                ui_messages::UiMessageHelper::new(app_handle, app_name, window_helper)
                    .listen(rx)
                    .await;
            });

            info!("Setup done");
            Ok(())
        })
        .on_system_tray_event(system_tray_event_handler);

    #[cfg(feature = "enable-updater")]
    {
        app_builder = app_builder.invoke_handler(tauri::generate_handler![
            ui_ready::ui_ready,
            action_logs::write_action_log,
            action_logs::get_action_logs,
            action_logs::sync_action_logs,
            install_cli::install_cli,
            community_contributions::get_contributions,
            updates::get_releases,
            updates::get_pending_update,
            updates::check_updates
        ]);
    }
    #[cfg(not(feature = "enable-updater"))]
    {
        app_builder = app_builder.invoke_handler(tauri::generate_handler![
            ui_ready::ui_ready,
            action_logs::write_action_log,
            action_logs::get_action_logs,
            action_logs::sync_action_logs,
            install_cli::install_cli,
            community_contributions::get_contributions,
        ]);
    }

    let app = app_builder
        .build(ctx)
        .expect("error while building tauri application");

    info!("Run");

    #[cfg(feature = "enable-updater")]
    let config = app.config();
    app.run(move |app_handle, event| {
        let exit_requested_tx = tx.clone();

        match event {
            #[cfg(feature = "enable-updater")]
            tauri::RunEvent::Updater(updater_event) => {
                let app_handle = app_handle.clone();
                let app_identifier = config.tauri.bundle.identifier.clone();
                tauri::async_runtime::spawn(async move {
                    updates::UpdateHelper::new(&app_handle)
                        .handle_event(updater_event, &app_identifier)
                        .await;
                });
            }
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
                        let window_helper = window::WindowHelper::new(app_handle.clone());
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
