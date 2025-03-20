use tauri::AppHandle;
use crate::resource_watcher::ProInstance;

use super::{
    config::{CommandConfig, DevpodCommandConfig, DevpodCommandError},
    constants::{DEVPOD_BINARY_NAME, DEVPOD_COMMAND_LIST, DEVPOD_COMMAND_PRO, FLAG_OUTPUT_JSON},
};

pub struct ListProInstancesCommand {}
impl ListProInstancesCommand {
    pub fn new() -> Self {
        ListProInstancesCommand {}
    }

    fn deserialize(&self, d: Vec<u8>) -> Result<Vec<ProInstance>, DevpodCommandError> {
        serde_json::from_slice(&d).map_err(DevpodCommandError::Parse)
    }
}
impl DevpodCommandConfig<Vec<ProInstance>> for ListProInstancesCommand {
    fn config(&self) -> CommandConfig {
        CommandConfig {
            binary_name: DEVPOD_BINARY_NAME,
            args: vec![DEVPOD_COMMAND_PRO, DEVPOD_COMMAND_LIST, FLAG_OUTPUT_JSON],
        }
    }

    fn exec_blocking(self, app_handle: &AppHandle) -> Result<Vec<ProInstance>, DevpodCommandError> {
        let cmd = self.new_command(app_handle)?;

        let output = tauri::async_runtime::block_on(async move { cmd.output().await })
            .map_err(|_| DevpodCommandError::Output)?;

        self.deserialize(output.stdout)
    }
}
impl ListProInstancesCommand {
    pub async fn exec(
        self,
        app_handle: &AppHandle,
    ) -> Result<Vec<ProInstance>, DevpodCommandError> {
        let cmd = self.new_command(app_handle)?;

        let output = cmd.output().await.map_err(|_| DevpodCommandError::Output)?;

        self.deserialize(output.stdout)
    }
}
