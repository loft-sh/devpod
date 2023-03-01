use crate::workspaces::WorkspacesState;

use super::{
    config::{CommandConfig, DevpodCommandConfig, DevpodCommandError},
    constants::{DEVPOD_BINARY_NAME, DEVPOD_COMMAND_LIST, OUTPUT_JSON_ARG},
};

pub struct ListWorkspacesCommand {}
impl ListWorkspacesCommand {
    pub fn new() -> Self {
        ListWorkspacesCommand {}
    }
}
impl DevpodCommandConfig<WorkspacesState> for ListWorkspacesCommand {
    fn config(&self) -> CommandConfig {
        CommandConfig {
            binary_name: DEVPOD_BINARY_NAME,
            args: vec![DEVPOD_COMMAND_LIST, OUTPUT_JSON_ARG],
        }
    }

    fn deserialize(&self, str: &str) -> Result<WorkspacesState, DevpodCommandError> {
        serde_json::from_str(str).map_err(|err| DevpodCommandError::Parse(err))
    }
}
