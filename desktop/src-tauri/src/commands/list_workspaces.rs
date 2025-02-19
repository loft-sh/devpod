use tauri::AppHandle;

use crate::resource_watcher::Workspace;

use super::{
    config::{CommandConfig, DevpodCommandConfig, DevpodCommandError},
    constants::{DEVPOD_BINARY_NAME, DEVPOD_COMMAND_LIST, FLAG_OUTPUT_JSON},
};

pub struct ListWorkspacesCommand {}
impl ListWorkspacesCommand {
    pub fn new() -> Self {
        ListWorkspacesCommand {}
    }

    fn deserialize(&self, d: Vec<u8>) -> Result<Vec<Workspace>, DevpodCommandError> {
        serde_json::from_slice(&d).map_err(DevpodCommandError::Parse)
    }
}
impl DevpodCommandConfig<Vec<Workspace>> for ListWorkspacesCommand {
    fn config(&self) -> CommandConfig {
        CommandConfig {
            binary_name: DEVPOD_BINARY_NAME,
            args: vec![DEVPOD_COMMAND_LIST, FLAG_OUTPUT_JSON],
        }
    }

    fn exec_blocking(self, app_handle: &AppHandle) -> Result<Vec<Workspace>, DevpodCommandError> {
        let cmd = self.new_command(app_handle)?;

        let output = tauri::async_runtime::block_on(async move { cmd.output().await })
            .map_err(|_| DevpodCommandError::Output)?;

        self.deserialize(output.stdout)
    }
}

impl ListWorkspacesCommand {
    pub async fn exec(self, app_handle: &AppHandle) -> Result<Vec<Workspace>, DevpodCommandError> {
        let cmd = self.new_command(app_handle)?;

        let output = cmd.output().await.map_err(|_| DevpodCommandError::Output)?;

        self.deserialize(output.stdout)
    }
}
