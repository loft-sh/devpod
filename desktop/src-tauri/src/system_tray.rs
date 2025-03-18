use crate::{
    resource_watcher::{ProState, WorkspacesState},
    ui_messages::{OpenProInstanceMsg, OpenWorkspaceMsg},
    util, AppHandle, AppState, UiMessage,
};
use log::{error, warn};
use tauri::{
    menu::{Menu, MenuBuilder, MenuEvent, MenuItem, Submenu},
    tray::{MouseButton, TrayIcon, TrayIconEvent},
    Manager, Wry,
};
use util::QUIT_EXIT_CODE;

#[cfg(not(target_os = "macos"))]
pub static WARNING_SYSTEM_TRAY_ICON_BYTES: &'static [u8] = include_bytes!("../icons/icon_warning_system_tray_color.png");
#[cfg(target_os = "macos")]
pub static WARNING_SYSTEM_TRAY_ICON_BYTES: &'static [u8] = include_bytes!("../icons/icon_warning_system_tray.png");

#[cfg(not(target_os = "macos"))]
pub static SYSTEM_TRAY_ICON_BYTES: &'static [u8] = include_bytes!("../icons/icon_system_tray_color.png");
#[cfg(target_os = "macos")]
pub static SYSTEM_TRAY_ICON_BYTES: &'static [u8] = include_bytes!("../icons/icon_system_tray.png");

pub trait ToSystemTraySubmenu {
    fn to_submenu(&self, app_handle: &AppHandle) -> anyhow::Result<Submenu<Wry>>;
}

pub struct SystemTray {}

impl SystemTray {
    pub fn new() -> Self {
        SystemTray {}
    }
}

impl SystemTray {
    const QUIT_ID: &str = "quit";
    const SHOW_DASHBOARD_ID: &str = "show_dashboard";
}

impl SystemTray {
    pub async fn init(&self, app_handle: &AppHandle) -> anyhow::Result<Menu<Wry>> {
        let mut menu = MenuBuilder::new(app_handle);
        let show_dashboard = MenuItem::with_id(
            app_handle,
            Self::SHOW_DASHBOARD_ID,
            "Show Dashboard",
            true,
            None::<&str>,
        )?;
        menu = menu.item(&show_dashboard);

        let state = app_handle.state::<AppState>();

        let mut workspaces = state.workspaces.write().await;
        let submenu = workspaces.to_submenu(app_handle)?;
        menu = menu.item(&submenu);
        workspaces.set_submenu(submenu);

        let mut pro = state.pro.write().await;
        let submenu = pro.to_submenu(app_handle)?;
        menu = menu.item(&submenu);
        pro.set_submenu(submenu);

        let quit = MenuItem::with_id(app_handle, Self::QUIT_ID, "Quit", true, None::<&str>)?;
        menu = menu.item(&quit);

        let m = menu.build()?;

        Ok(m)
    }

    pub fn get_menu_event_handler(&self) -> impl Fn(&AppHandle, MenuEvent) + Send + Sync {
        |app, event| match event.id.as_ref() {
            Self::QUIT_ID => app.exit(QUIT_EXIT_CODE),
            Self::SHOW_DASHBOARD_ID => {
                let app_state = app.state::<AppState>();

                tauri::async_runtime::block_on(async move {
                    if let Err(err) = app_state.ui_messages.send(UiMessage::ShowDashboard).await {
                        error!("Failed to broadcast show dashboard message: {}", err);
                    };
                });
            }
            id => {
                let app_state = app.state::<AppState>();

                tauri::async_runtime::block_on(async move {
                    if let Err(err) = app_state.ui_messages.send(UiMessage::ShowDashboard).await {
                        error!("Failed to broadcast show dashboard message: {}", err);
                    };
                    if id.starts_with(WorkspacesState::IDENTIFIER_PREFIX) {
                        let tx = &app_state.ui_messages;

                        if id == WorkspacesState::CREATE_WORKSPACE_ID {
                            if let Err(err) = tx
                                .send(UiMessage::OpenWorkspace(OpenWorkspaceMsg::empty()))
                                .await
                            {
                                error!("Failed to send create workspace message: {:?}", err);
                            };
                        } else {
                            let workspace_id = id.replace(WorkspacesState::IDENTIFIER_PREFIX, "");
                            if let Err(err) = tx
                                .send(UiMessage::OpenWorkspace(OpenWorkspaceMsg::with_id(
                                    workspace_id,
                                )))
                                .await
                            {
                                error!("Failed to send create workspace message: {:?}", err);
                            };
                        }
                    } else if id.starts_with(ProState::IDENTIFIER_PREFIX) {
                        let tx = &app_state.ui_messages;

                        let host = id.replace(ProState::IDENTIFIER_PREFIX, "");
                        if let Err(err) = tx
                            .send(UiMessage::OpenProInstance(OpenProInstanceMsg {
                                host: Some(host),
                            }))
                            .await
                        {
                            error!("Failed to send open pro instance message: {:?}", err);
                        };
                    } else {
                        warn!("Received unhandled click for ID: {}", id);
                    }
                });
            }
        }
    }

    pub fn get_tray_icon_event_handler(&self) -> impl Fn(&TrayIcon, TrayIconEvent) + Send + Sync {
        |icon, event| match event {
            TrayIconEvent::DoubleClick { button, .. } => {
                if button == MouseButton::Left {
                    let app_state = icon.app_handle().state::<AppState>();

                    tauri::async_runtime::block_on(async move {
                        if let Err(err) = app_state.ui_messages.send(UiMessage::ShowDashboard).await
                        {
                            error!("Failed to broadcast show dashboard message: {}", err);
                        };
                    });
                }
            }
            _ => {}
        }
    }
}
