use std::collections::HashMap;

use tauri::AppHandle;
use tauri_plugin_shell::{process::Command, ShellExt};
use thiserror::Error;

use crate::commands::constants::DEVPOD_BINARY_NAME;

use super::constants::DEVPOD_UI_ENV_VAR;

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
    Failed(#[from] tauri_plugin_shell::Error),
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
    fn exec(self, app_handle: &AppHandle) -> Result<T, DevpodCommandError>;

    fn new_command(&self, app_handle: &AppHandle) -> Result<Command, DevpodCommandError> {
        let config = self.config();
        let env_vars: HashMap<String, String> =
            HashMap::from([(DEVPOD_UI_ENV_VAR.into(), "true".into())]);

        let cmd = app_handle
            .shell()
            .sidecar(config.binary_name())
            .map_err(|_| DevpodCommandError::Sidecar)?
            .envs(env_vars)
            .args(config.args());

        Ok(cmd)
    }
}
