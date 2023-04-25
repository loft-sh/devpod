use crate::{commands::DEVPOD_BINARY_NAME, AppHandle};
use std::{
    env,
    path::{Path, PathBuf},
};
use thiserror::Error;

#[derive(Error, Debug)]
pub enum InstallCLIError {
    #[error("Unable to get current executable path")]
    NoExePath(#[source] std::io::Error),
    #[error("Unable to create symlink")]
    Symlink(#[source] std::io::Error),
    #[error("Unable to convert path to string")]
    PathConversion,
    #[error("Encountered an issue with the windows registry: ")]
    Registry(#[source] std::io::Error),
    #[error("No data directory found")]
    DataDir,
    #[error("Unable to create directory")]
    CreateDir(#[source] std::io::Error),
    #[error("Unable to write to file")]
    WriteFile(#[source] std::io::Error),
    #[error("Failed to inform Windows about the change in environment variables. You will need to reboot you machine for them to take effect.")]
    WindowsBroadcastChange,
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
pub fn install_cli(app_handle: AppHandle) -> Result<(), InstallCLIError> {
    install(app_handle)
}

// The path to the `devpod-cli` binary/executable. If bundled correctly, will be placed next to the desktop app executable.
fn get_cli_path() -> Result<PathBuf, std::io::Error> {
    let mut exe_path = env::current_exe()?;
    exe_path.pop();
    exe_path.push(DEVPOD_BINARY_NAME);

    Ok(exe_path)
}

#[cfg(not(target_os = "windows"))]
fn install(_app_handle: AppHandle) -> Result<(), InstallCLIError> {
    use std::{fs::remove_file, os::unix::fs::symlink};

    let cli_path = get_cli_path().map_err(|e| InstallCLIError::NoExePath(e))?;

    // The binary we ship with is `devpod-cli`, but we want to symlink it to `devpod` so that users can just run `devpod` in their terminal
    let raw_target_path = format!("/usr/local/bin/{}", "devpod");
    let target_path = Path::new(&raw_target_path);

    match target_path.try_exists() {
        Ok(..) => {
            // remove symlink first before attempting to create another one
            remove_file(target_path).map_err(|e| InstallCLIError::Symlink(e))?;
        }
        _ => { /* fallthrough */ }
    }

    symlink(cli_path, target_path).map_err(|e| InstallCLIError::Symlink(e))
}

#[cfg(target_os = "windows")]
fn install(app_handle: AppHandle) -> Result<(), InstallCLIError> {
    use log::error;
    use std::fs;
    use windows::Win32::{
        Foundation::{GetLastError, HWND, LPARAM},
        UI::WindowsAndMessaging::{SendMessageTimeoutW, SMTO_ABORTIFHUNG, WM_SETTINGCHANGE},
    };
    use winreg::{
        enums::{HKEY_CURRENT_USER, KEY_ALL_ACCESS},
        RegKey,
    };

    struct BinFile {
        name: String,
        content: String,
    }

    let cli_path = get_cli_path().map_err(|e| InstallCLIError::NoExePath(e))?;
    let mut bin_dir = app_handle
        .path_resolver()
        .app_data_dir()
        .ok_or(InstallCLIError::DataDir)?;
    bin_dir.push("bin");

    // Create binary directory in app dir and write bin_files to disk
    // These will be stored in a /bin folder under our control, usually `%APP_DIR%/sh.loft.desktop-desktop/bin`
    let cli_path = cli_path.to_str().ok_or(InstallCLIError::PathConversion)?;

    let sh_file = BinFile {
        name: "devpod".to_string(),
        // WARN: we actually need to debug print here because this escapes the backslash to `\\` and will then be recognised by the shell
        content: format!("#!/usr/bin/env sh\n{:?}.exe\nexit $?", cli_path),
    };

    let cmd_file = BinFile {
        name: format!("{}.cmd", "devpod".to_string()),
        content: format!("@echo off\n\"{}.exe\"", cli_path),
    };

    fs::create_dir_all(bin_dir.clone()).map_err(|e| InstallCLIError::CreateDir(e))?;
    for BinFile { content, name } in [sh_file, cmd_file] {
        let mut file_path = bin_dir.clone();
        file_path.push(name);

        if let Err(e) = fs::write(file_path, content.as_bytes()) {
            return Err(InstallCLIError::WriteFile(e));
        }
    }

    // Now that we placed our entry points in the /bin folder, we need to update the users path environment variable
    // to include said folder
    let current_dir_path = bin_dir.to_str().ok_or(InstallCLIError::PathConversion)?;
    let hkcu = RegKey::predef(HKEY_CURRENT_USER);
    let environment_key = hkcu
        .open_subkey_with_flags("Environment", KEY_ALL_ACCESS)
        .map_err(|e| InstallCLIError::Registry(e))?;
    let mut current_env_path: String = environment_key
        .get_value("Path")
        .map_err(|e| InstallCLIError::Registry(e))?;

    // Make sure we only add the path once
    if current_env_path.contains(current_dir_path) {
        return Ok(());
    }

    current_env_path.push_str(&format!(";{}", current_dir_path));

    environment_key
        .set_value("Path", &current_env_path)
        .map_err(|e| InstallCLIError::Registry(e))?;

    // After setting the registry key we need to inform windows about the changes.
    // Otherwise it would require a full system reboot for them to take effect.
    unsafe {
        // See https://learn.microsoft.com/en-us/windows/win32/winmsg/wm-settingchange
        // and https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendmessagetimeoutw
        // for more information about `WM_SETTINGCHANGE` and `SendMessageTimeoutW`.

        #[allow(non_snake_case)]
        let HWND_BROADCAST = HWND(0xffff);
        let environment_ptr = LPARAM("Environment".as_ptr() as isize);
        // Apparently we can only broadcast this message via a synchronous operation, `PostMessage` doesn't work here
        // This function blocks until either all windows handled the message or the timeout is exceeded for every one of them. You can check the documentation for details.
        // Because of this, we need to ensure this function will only be called on a thread that's okay with being blocked for some time.
        // Right now this will only be called from a tauri command, so we're good. If you need to call this function from somewhere else, be aware of the blocking.
        let result = SendMessageTimeoutW(
            HWND_BROADCAST,
            WM_SETTINGCHANGE,
            None,
            environment_ptr,
            SMTO_ABORTIFHUNG,
            3_000,
            None,
        );
        if result.0 == 0 {
            let last_error = GetLastError();
            error!("{:?}", last_error);

            return Err(InstallCLIError::WindowsBroadcastChange);
        }
    };

    Ok(())
}
