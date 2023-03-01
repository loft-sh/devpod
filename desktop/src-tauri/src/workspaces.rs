use crate::{
    commands::{list_workspaces::ListWorkspacesCommand, DevpodCommandConfig, DevpodCommandError},
    system_tray::ToSystemTraySubmenu,
};
use chrono::DateTime;
use log::trace;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use tauri::{api::process::Command, CustomMenuItem, SystemTrayMenu, SystemTraySubmenu};

#[derive(Serialize, Deserialize, Debug)]
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
        let config = list_workspaces_cmd.config();

        // TODO: maybe refactor into `Command` type
        let output = Command::new_sidecar(config.binary_name())
            .expect("should have found `devpod` binary")
            .args(config.args())
            .output()
            .expect("should have spawned `devpod`");

        list_workspaces_cmd.deserialize(&output.stdout)
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
}

#[derive(Serialize, Deserialize, Debug)]
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

#[derive(Serialize, Deserialize, Debug)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
struct WorkspaceProvider {
    name: Option<String>,
}

#[derive(Serialize, Deserialize, Debug)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
struct WorkspaceIDE {
    #[serde(rename = "ide")]
    id: Option<String>,
    options: Option<HashMap<String, String>>,
}

#[derive(Serialize, Deserialize, Debug)]
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
