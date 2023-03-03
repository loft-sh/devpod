use crate::{
    commands::{list_workspaces::ListWorkspacesCommand, DevpodCommandConfig, DevpodCommandError},
    system_tray::{SystemTrayClickHandler, ToSystemTraySubmenu},
};
use chrono::DateTime;
use log::trace;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use tauri::{CustomMenuItem, SystemTrayMenu, SystemTraySubmenu};

#[derive(Serialize, Deserialize, Debug, Default, Eq, PartialEq)]
#[serde(
    transparent,
    rename_all(serialize = "camelCase", deserialize = "camelCase")
)]
pub struct WorkspacesState {
    workspaces: Vec<Workspace>,
}
impl WorkspacesState {
    pub const IDENTIFIER_PREFIX: &str = "workspaces";

    fn item_id(id: &String) -> String {
        format!("{}-{}", Self::IDENTIFIER_PREFIX, id)
    }
}
impl WorkspacesState {
    pub fn load() -> Result<Self, DevpodCommandError> {
        trace!("loading workspaces");
        let list_workspaces_cmd = ListWorkspacesCommand::new();

        list_workspaces_cmd.exec()
    }
}
impl ToSystemTraySubmenu for WorkspacesState {
    fn to_submenu(&self) -> tauri::SystemTraySubmenu {
        let mut providers_menu = SystemTrayMenu::new();
        for workspace in &self.workspaces {
            if let Some(id) = workspace.id() {
                let item = CustomMenuItem::new(Self::item_id(id), id);
                providers_menu = providers_menu.add_item(item);
            }
        }

        SystemTraySubmenu::new("Workspaces", providers_menu)
    }

    fn on_tray_item_clicked(&self, _id: &str) -> Option<SystemTrayClickHandler> {
        todo!()
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
