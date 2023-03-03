use log::warn;
use tauri::{
    AppHandle, CustomMenuItem, Manager, State, SystemTray as TauriSystemTray, SystemTrayEvent,
    SystemTrayMenu, SystemTrayMenuItem, SystemTraySubmenu, WindowBuilder, WindowUrl, Wry,
};

use crate::{providers::ProvidersState, workspaces::WorkspacesState, AppState};

pub trait SystemTrayIdentifier {}
pub type SystemTrayClickHandler = Box<dyn Fn(&AppHandle, State<AppState>) -> ()>;
pub trait ToSystemTraySubmenu {
    fn to_submenu(&self) -> SystemTraySubmenu;
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

// FIXME: should implement proper builder pattern
impl SystemTray {
    pub fn build_menu(&self) -> SystemTrayMenu {
        let show_dashboard = CustomMenuItem::new(Self::SHOW_DASHBOARD_ID, "Show Dashboard");
        let quit = CustomMenuItem::new(Self::QUIT_ID, "Quit");

        let tray_menu = SystemTrayMenu::new()
            .add_item(show_dashboard)
            .add_native_item(SystemTrayMenuItem::Separator)
            .add_item(quit);

        tray_menu
    }

    pub fn build_with_submenus(
        &self,
        submenu_builders: Vec<Box<&dyn ToSystemTraySubmenu>>,
    ) -> SystemTrayMenu {
        let show_dashboard = CustomMenuItem::new(Self::SHOW_DASHBOARD_ID, "Show Dashboard");
        let quit = CustomMenuItem::new(Self::QUIT_ID, "Quit");

        let mut tray_menu = SystemTrayMenu::new()
            .add_item(show_dashboard)
            .add_native_item(SystemTrayMenuItem::Separator);

        for builder in submenu_builders {
            let submenu = builder.to_submenu();
            tray_menu = tray_menu.add_submenu(submenu)
        }

        tray_menu = tray_menu
            .add_native_item(SystemTrayMenuItem::Separator)
            .add_item(quit);

        tray_menu
    }

    pub fn build(&self) -> TauriSystemTray {
        let tray_menu = self.build_menu();
        let tray = TauriSystemTray::new().with_menu(tray_menu);

        tray
    }

    pub fn get_event_handler(&self) -> impl Fn(&AppHandle<Wry>, SystemTrayEvent) + Send + Sync {
        return |app, event| match event {
            SystemTrayEvent::MenuItemClick { id, .. } => match id.as_str() {
                Self::QUIT_ID => {
                    std::process::exit(0);
                }
                Self::SHOW_DASHBOARD_ID => {
                    match app.get_window("main") {
                        Some(window) => {
                            _ = window.show(); // TODO: handle error
                        }
                        None => {
                            // FIXME: implement correctly and reread from original window
                            _ = WindowBuilder::new(app, "main".to_string(), WindowUrl::default())
                                .title("Main")
                                .build();
                        }
                    }
                }
                id => {
                    let app_state = app.state::<AppState>();
                    let mut maybe_handler: Option<_> = None;

                    if id.starts_with(WorkspacesState::IDENTIFIER_PREFIX) {
                        let workspaces_state = &*app_state.workspaces.lock().unwrap();
                        maybe_handler = workspaces_state.on_tray_item_clicked(id);
                    } else if id.starts_with(ProvidersState::IDENTIFIER_PREFIX) {
                        let providers_state = &*app_state.providers.lock().unwrap();
                        maybe_handler = providers_state.on_tray_item_clicked(id);
                    } else {
                        warn!("Received unhandled click for ID: {}", id);
                    }

                    if let Some(handler) = maybe_handler {
                        handler(app, app_state);

                        return;
                    }

                    return;
                }
            },
            _ => {}
        };
    }
}
