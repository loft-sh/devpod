use crate::{
    commands::{list_workspaces::ListWorkspacesCommand, DevpodCommandConfig, DevpodCommandError},
    custom_protocol::OpenWorkspaceMsg,
    system_tray::{SystemTrayClickHandler, ToSystemTraySubmenu},
};
use crate::{system_tray::SystemTray, AppHandle, AppState, UiMessage};
use chrono::DateTime;
use log::error;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::{
    sync::{mpsc, Arc},
    thread, time,
};
use tauri::{CustomMenuItem, SystemTrayMenu, SystemTrayMenuItem, SystemTraySubmenu};
use tokio::sync::OnceCell;

static INIT: OnceCell<()> = OnceCell::const_new();

enum Update {
    Workspaces(WorkspacesState),
}

#[derive(Serialize, Deserialize, Debug, Default, Eq, PartialEq)]
#[serde(
    transparent,
    rename_all(serialize = "camelCase", deserialize = "camelCase")
)]
pub struct WorkspacesState {
    workspaces: Vec<Workspace>,
}

impl WorkspacesState {
    pub const IDENTIFIER_PREFIX: &str = "workspaces-";
    const CREATE_WORKSPACE_ID: &str = "workspaces-create_workspace";

    fn item_id(id: &String) -> String {
        format!("{}{}", Self::IDENTIFIER_PREFIX, id)
    }
}

impl WorkspacesState {
    pub fn load() -> Result<Self, DevpodCommandError> {
        let list_workspaces_cmd = ListWorkspacesCommand::new();

        list_workspaces_cmd.exec()
    }
}

impl WorkspacesState {}

impl ToSystemTraySubmenu for WorkspacesState {
    fn to_submenu(&self) -> tauri::SystemTraySubmenu {
        let mut workspaces_menu = SystemTrayMenu::new();

        workspaces_menu = workspaces_menu.add_item(CustomMenuItem::new(
            Self::CREATE_WORKSPACE_ID,
            "Create Workspace",
        ));

        if !self.workspaces.is_empty() {
            workspaces_menu = workspaces_menu.add_native_item(SystemTrayMenuItem::Separator);
        }

        for workspace in &self.workspaces {
            if let Some(id) = workspace.id() {
                let item = CustomMenuItem::new(Self::item_id(id), id);
                workspaces_menu = workspaces_menu.add_item(item);
            }
        }

        SystemTraySubmenu::new("Workspaces", workspaces_menu)
    }

    fn on_tray_item_clicked(&self, id: &str) -> Option<SystemTrayClickHandler> {
        let id = id.clone().to_string();

        Some(Box::new(move |_app_handle, state| {
            tauri::async_runtime::block_on(async {
                let tx = &state.ui_messages;

                if id == Self::CREATE_WORKSPACE_ID {
                    if let Err(err) = tx
                        .send(UiMessage::OpenWorkspace(OpenWorkspaceMsg::empty()))
                        .await
                    {
                        error!("Failed to send create workspace message: {:?}", err);
                    };
                } else {
                    let workspace_id = id.replace(Self::IDENTIFIER_PREFIX, "");
                    if let Err(err) = tx
                        .send(UiMessage::OpenWorkspace(OpenWorkspaceMsg::with_id(
                            workspace_id,
                        )))
                        .await
                    {
                        error!("Failed to send create workspace message: {:?}", err);
                    };
                }
            })
        }))
    }
}

#[derive(Serialize, Deserialize, Debug, Eq, PartialEq)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
struct Workspace {
    id: Option<String>,
    folder: Option<String>,
    provider: Option<WorkspaceProvider>,
    #[serde(rename = "ide")]
    ide_config: Option<WorkspaceIDE>,
    source: Option<WorkspaceSource>,
    creation_timestamp: Option<chrono::DateTime<chrono::Utc>>,
    context: Option<String>,
}
impl Workspace {
    pub fn id(&self) -> &Option<String> {
        &self.id
    }
}

#[derive(Serialize, Deserialize, Debug, Eq, PartialEq)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
struct WorkspaceProvider {
    name: Option<String>,
}

#[derive(Serialize, Deserialize, Debug, Eq, PartialEq)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
struct WorkspaceIDE {
    #[serde(rename = "ide")]
    id: Option<String>,
    options: Option<HashMap<String, String>>,
}

#[derive(Serialize, Deserialize, Debug, Eq, PartialEq)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
struct WorkspaceSource {
    git_repository: Option<String>,
    git_branch: Option<String>,
    git_commit: Option<String>,
    local_folder: Option<String>,
    image: Option<String>,
}

#[derive(Serialize, Deserialize, Debug)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
struct ProviderOption {
    value: Option<String>,
    local: Option<bool>,
    retrieved: Option<DateTime<chrono::Utc>>,
}

pub fn setup(app_handle: &AppHandle, state: tauri::State<'_, AppState>) {
    tauri::async_runtime::block_on(async {
        INIT.get_or_init(|| async {
            let sleep_duration = time::Duration::from_millis(1_000);
            let (tx, rx) = mpsc::channel::<Update>();

            let workspaces_tx = tx;

            thread::spawn(move || loop {
                let workspaces = WorkspacesState::load().unwrap();
                workspaces_tx.send(Update::Workspaces(workspaces)).unwrap();

                thread::sleep(sleep_duration);
            });

            let workspaces_state = Arc::clone(&state.workspaces);
            let tray_handle = app_handle.tray_handle();

            // Handle updates from background threads.
            thread::spawn(move || {
                while let Ok(msg) = rx.recv() {
                    match msg {
                        Update::Workspaces(workspaces) => {
                            let current_workspaces = &mut *workspaces_state.lock().unwrap();

                            if current_workspaces != &workspaces {
                                *current_workspaces = workspaces;

                                // rebuild menu
                                let new_menu = SystemTray::new()
                                    .build_menu(vec![Box::new(current_workspaces)]);
                                tray_handle
                                    .set_menu(new_menu)
                                    .expect("should be able to set menu");
                            }
                        }
                    }
                }
            });
        })
        .await;
    });
}
