use crate::{util, workspaces::WorkspacesState, AppHandle, AppState, UiMessage};
use log::{error, warn};
use tauri::{
    menu::{Menu, MenuBuilder, MenuEvent, MenuItem, Submenu, SubmenuBuilder},
    tray::{TrayIcon, TrayIconEvent},
    EventLoopMessage, Manager, State, Wry,
};
use util::QUIT_EXIT_CODE;

pub trait SystemTrayIdentifier {}
pub type SystemTrayClickHandler = Box<dyn Fn(&AppHandle, State<AppState>)>;
pub trait ToSystemTraySubmenu {
    fn to_submenu(&self, app_handle: &AppHandle) -> anyhow::Result<Submenu<Wry>>;
    fn on_tray_item_clicked(&self, tray_item_id: &str) -> Option<SystemTrayClickHandler>;
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
// let show_dashboard =
//     MenuItem::with_id(app, "show_dashboard", "Show Dashboard", true, None::<&str>)?;
// let quit = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
// let menu = Menu::with_items(app, &[&show_dashboard, &quit])?;

impl SystemTray {
    pub fn build_menu(
        &self,
        app_handle: &AppHandle,
        builder: Box<&dyn ToSystemTraySubmenu>,
    ) -> anyhow::Result<Menu<Wry>> {
        let mut menu = MenuBuilder::new(app_handle);
        let show_dashboard = MenuItem::with_id(
            app_handle,
            Self::SHOW_DASHBOARD_ID,
            "Show Dashboard",
            true,
            None::<&str>,
        )?;
        menu = menu.item(&show_dashboard);

        let submenu = builder.to_submenu(app_handle)?;
        menu = menu.item(&submenu);

        let quit = MenuItem::with_id(app_handle, Self::QUIT_ID, "Quit", true, None::<&str>)?;
        menu = menu.item(&quit);

        let m = menu.build()?;

        Ok(m)
    }

    // pub fn build_tray(
    //     &self,
    //     submenu_builders: Vec<Box<&dyn ToSystemTraySubmenu>>,
    // ) -> TauriSystemTray {
    //     let tray_menu = self.build_menu(submenu_builders);
    //
    //     TauriSystemTray::new().with_menu(tray_menu)
    // }

    pub fn get_event_handler(&self) -> impl Fn(&AppHandle, MenuEvent) + Send + Sync {
        |app, event| match event.id.as_ref() {
            Self::QUIT_ID => {
                app.exit(QUIT_EXIT_CODE)
            }
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
                let mut maybe_handler: Option<_> = None;

                if id.starts_with(WorkspacesState::IDENTIFIER_PREFIX) {
                    let workspaces_state = &*app_state.workspaces.lock().unwrap();
                    maybe_handler = workspaces_state.on_tray_item_clicked(id);
                } else {
                    warn!("Received unhandled click for ID: {}", id);
                }

                if let Some(handler) = maybe_handler {
                    handler(app, app_state);
                }
            }
        }
    }
}
