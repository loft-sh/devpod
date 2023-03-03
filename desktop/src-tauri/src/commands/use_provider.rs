use super::{
    config::{CommandConfig, DevpodCommandConfig, DevpodCommandError},
    constants::{DEVPOD_BINARY_NAME, DEVPOD_COMMAND_PROVIDER, DEVPOD_COMMAND_USE},
};

pub struct UseProviderCommand<'a> {
    new_provider_id: &'a str,
}
impl<'a> UseProviderCommand<'a> {
    pub fn new(new_provider_id: &'a str) -> Self {
        UseProviderCommand { new_provider_id }
    }
}
impl DevpodCommandConfig<()> for UseProviderCommand<'_> {
    fn config(&self) -> CommandConfig {
        CommandConfig {
            binary_name: DEVPOD_BINARY_NAME,
            args: vec![
                DEVPOD_COMMAND_PROVIDER,
                DEVPOD_COMMAND_USE,
                self.new_provider_id,
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
