use crate::providers::ProvidersState;

use super::{
    config::{CommandConfig, DevpodCommandConfig, DevpodCommandError},
    constants::{
        DEVPOD_BINARY_NAME, DEVPOD_COMMAND_LIST, DEVPOD_COMMAND_PROVIDERS, OUTPUT_JSON_ARG,
    },
};

pub struct ListProvidersCommand {}
impl ListProvidersCommand {
    pub fn new() -> Self {
        ListProvidersCommand {}
    }
}
impl DevpodCommandConfig<ProvidersState> for ListProvidersCommand {
     fn config(&self) -> CommandConfig {
        CommandConfig {
            binary_name: DEVPOD_BINARY_NAME,
            args: vec![
                DEVPOD_COMMAND_PROVIDERS,
                DEVPOD_COMMAND_LIST,
                OUTPUT_JSON_ARG,
            ],
        }
    }

    fn deserialize(&self, str: &str) -> Result<ProvidersState, DevpodCommandError> {
        serde_json::from_str(str).map_err(|err| DevpodCommandError::Parse(err))
    }
}
