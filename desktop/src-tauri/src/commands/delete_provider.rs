use super::{
    config::{CommandConfig, DevpodCommandConfig, DevpodCommandError},
    constants::{DEVPOD_BINARY_NAME, DEVPOD_COMMAND_PROVIDER, DEVPOD_COMMAND_USE, DEVPOD_COMMAND_DELETE},
};

pub struct DeleteProviderCommand {
    provider_id: String
}
impl DeleteProviderCommand {
    pub fn new(provider_id: String) -> Self {
        DeleteProviderCommand {
            provider_id
        }
    }
}
impl DevpodCommandConfig<()> for DeleteProviderCommand {
    fn config(&self) -> CommandConfig {
        CommandConfig {
            binary_name: DEVPOD_BINARY_NAME,
            args: vec![
                DEVPOD_COMMAND_PROVIDER,
                DEVPOD_COMMAND_DELETE,
                &self.provider_id,
            ],
        }
    }

    fn exec(self) -> Result<(), DevpodCommandError> {
        let cmd = self.new_command()?;

        cmd.status()
            .map_err(|err| DevpodCommandError::Failed(err))?
            .success()
            .then(|| ())
            .ok_or_else(|| DevpodCommandError::Exit)
    }
}
