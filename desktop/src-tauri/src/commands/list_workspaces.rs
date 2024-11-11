use tauri::AppHandle;

use super::{
    config::{CommandConfig, DevpodCommandConfig, DevpodCommandError},
    constants::{DEVPOD_BINARY_NAME, DEVPOD_COMMAND_LIST, FLAG_OUTPUT_JSON},
};
use crate::workspaces::WorkspacesState;

pub struct ListWorkspacesCommand {}
impl ListWorkspacesCommand {
    pub fn new() -> Self {
        ListWorkspacesCommand {}
    }

    fn deserialize(&self, d: Vec<u8>) -> Result<WorkspacesState, DevpodCommandError> {
        serde_json::from_slice(&d).map_err(DevpodCommandError::Parse)
    }
}
impl DevpodCommandConfig<WorkspacesState> for ListWorkspacesCommand {
    fn config(&self) -> CommandConfig {
        CommandConfig {
            binary_name: DEVPOD_BINARY_NAME,
            args: vec![DEVPOD_COMMAND_LIST, FLAG_OUTPUT_JSON],
        }
    }

    fn exec(self, app_handle: &AppHandle) -> Result<WorkspacesState, DevpodCommandError> {
        let cmd = self.new_command(app_handle)?;

        let output = tauri::async_runtime::block_on(async move { cmd.output().await })
            .map_err(|_| DevpodCommandError::Output)?;

        self.deserialize(output.stdout)
    }
}
