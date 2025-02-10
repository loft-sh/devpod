use serde::{Deserialize, Serialize};
use tauri::AppHandle;

use super::{
    config::{CommandConfig, DevpodCommandConfig, DevpodCommandError},
    constants::{DEVPOD_BINARY_NAME, DEVPOD_COMMAND_LIST, DEVPOD_COMMAND_PRO, FLAG_OUTPUT_JSON},
};

#[derive(Serialize, Deserialize, Debug, Eq, PartialEq)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
pub struct ProInstance {
    id: Option<String>,
    url: Option<String>,
    creation_timestamp: Option<chrono::DateTime<chrono::Utc>>,
}
impl ProInstance {
    pub fn id(&self) -> Option<&String> {
        self.id.as_ref()
    }
}

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

    fn exec(self, app_handle: &AppHandle) -> Result<Vec<ProInstance>, DevpodCommandError> {
        let cmd = self.new_command(app_handle)?;

        let output = tauri::async_runtime::block_on(async move { cmd.output().await })
            .map_err(|_| DevpodCommandError::Output)?;

        self.deserialize(output.stdout)
    }
}
