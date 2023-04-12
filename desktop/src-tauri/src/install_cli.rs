use std::{env, path::Path};

use anyhow::Context;
use thiserror::Error;

use crate::commands::DEVPOD_BINARY_NAME;

#[derive(Error, Debug)]
pub enum InstallCLIError {
    #[error("Platform not supported")]
    PlatformNotSupported,
    #[error("Unable to get current executable path")]
    NoExePath(#[source] std::io::Error),
    #[error("Unable to create symlink")]
    Symlink(#[source] std::io::Error),
}
impl serde::Serialize for InstallCLIError {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}

#[tauri::command]
pub fn install_cli() -> Result<(), InstallCLIError> {
    install()
}

#[cfg(not(target_os = "windows"))]
fn install() -> Result<(), InstallCLIError> {
    use std::{fs::remove_file, os::unix::fs::symlink};

    let mut exe_path = env::current_exe().map_err(|e| InstallCLIError::NoExePath(e))?;
    exe_path.pop();
    exe_path.push(DEVPOD_BINARY_NAME);

    let raw_target_path = format!("/usr/local/bin/{}", DEVPOD_BINARY_NAME);
    let target_path = Path::new(&raw_target_path);

    match target_path.try_exists() {
        Ok(..) => {
            // need to be remove first
            println!("exists, {}", target_path.display());
            remove_file(target_path).map_err(|e| InstallCLIError::Symlink(e))?;
        }
        _ => {
            println!("does not exist, {}", target_path.display());
        }
    }

    symlink(exe_path, target_path).map_err(|e| InstallCLIError::Symlink(e))
}

#[cfg(target_os = "windows")]
fn install() -> Result<(), InstallCLIError> {
    Err(InstallCLIError::PlatformNotSupported)
}
