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
    fn deserialize(&self, str: &str) -> Result<T, DevpodCommandError>;
}
