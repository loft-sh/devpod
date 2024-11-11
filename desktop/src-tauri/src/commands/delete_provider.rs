use tauri::AppHandle;

use super::{
    config::{CommandConfig, DevpodCommandConfig, DevpodCommandError},
    constants::{DEVPOD_BINARY_NAME, DEVPOD_COMMAND_DELETE, DEVPOD_COMMAND_PROVIDER},
};

pub struct DeleteProviderCommand {
    provider_id: String,
}
impl DeleteProviderCommand {
    pub fn new(provider_id: String) -> Self {
        DeleteProviderCommand { provider_id }
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

    fn exec(self, app_handle: &AppHandle) -> Result<(), DevpodCommandError> {
        let cmd = self.new_command(app_handle)?;

        tauri::async_runtime::block_on(async move { cmd.status().await })
            .map_err(DevpodCommandError::Failed)?
            .success()
            .then_some(())
            .ok_or_else(|| DevpodCommandError::Exit)
    }
}
