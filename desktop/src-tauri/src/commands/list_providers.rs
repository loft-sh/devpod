use super::{
    config::{CommandConfig, DevpodCommandConfig, DevpodCommandError},
    constants::{
        DEVPOD_BINARY_NAME, DEVPOD_COMMAND_LIST, DEVPOD_COMMAND_PROVIDER, OUTPUT_JSON_ARG,
    },
};
use crate::providers::ProvidersState;

pub struct ListProvidersCommand {}
impl ListProvidersCommand {
    pub fn new() -> Self {
        ListProvidersCommand {}
    }

    fn deserialize(&self, str: &str) -> Result<ProvidersState, DevpodCommandError> {
        serde_json::from_str(str).map_err(|err| DevpodCommandError::Parse(err))
    }
}
impl DevpodCommandConfig<ProvidersState> for ListProvidersCommand {
    fn config(&self) -> CommandConfig {
        CommandConfig {
            binary_name: DEVPOD_BINARY_NAME,
            args: vec![
                DEVPOD_COMMAND_PROVIDER,
                DEVPOD_COMMAND_LIST,
                OUTPUT_JSON_ARG,
            ],
        }
    }

    fn exec(self) -> Result<ProvidersState, DevpodCommandError> {
        let output = self
            .new_command()?
            .output()
            .map_err(|_| DevpodCommandError::Output)?;

        self.deserialize(&output.stdout)
    }
}
