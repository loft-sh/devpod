use tauri::api::process::Command;
use thiserror::Error;

use crate::commands::constants::DEVPOD_BINARY_NAME;

pub struct CommandConfig<'a> {
    pub(crate) binary_name: &'static str,
    pub(crate) args: Vec<&'a str>,
}

impl<'a> CommandConfig<'_> {
    pub fn binary_name(&self) -> &'static str {
        self.binary_name
    }

    pub fn args(&self) -> &Vec<&str> {
        &self.args
    }
}

#[derive(Error, Debug)]
pub enum DevpodCommandError {
    #[error("unable to parse command response")]
    Parse(#[from] serde_json::Error),
    #[error("unable to find sidecar binary")]
    Sidecar,
    #[error("unable to collect output from command")]
    Output,
    #[error("command failed")]
    Failed(#[from] tauri::api::Error),
    #[error("command exited with non-zero code")]
    Exit,
}
impl serde::Serialize for DevpodCommandError {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}
pub trait DevpodCommandConfig<T> {
    fn config(&self) -> CommandConfig {
        CommandConfig {
            binary_name: DEVPOD_BINARY_NAME,
            args: vec![],
        }
    }
    fn exec(self) -> Result<T, DevpodCommandError>;

    fn new_command(&self) -> Result<Command, DevpodCommandError> {
        let config = self.config();

        let cmd = Command::new_sidecar(config.binary_name())
            .map_err(|_| DevpodCommandError::Sidecar)?
            .args(config.args());

        Ok(cmd)
    }
}
