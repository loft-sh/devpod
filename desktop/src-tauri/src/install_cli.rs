use crate::{commands::DEVPOD_BINARY_NAME, AppHandle};
use log::error;
use std::path::Path;
use std::str::Lines;
use std::{env, path::PathBuf};
use thiserror::Error;

#[derive(Error, Debug)]
#[allow(dead_code)]
pub enum InstallCLIError {
    #[error("Unable to get current executable path")]
    NoExePath(#[source] std::io::Error),
    #[error("Unable to create link to cli {0}")]
    Link(#[source] anyhow::Error),
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
pub fn install_cli(app_handle: AppHandle, force: bool) -> Result<(), InstallCLIError> {
    if let Err(err) = install(app_handle, force) {
        error!("{}", err);
        Err(err)
    } else {
        Ok(())
    }
}

// The path to the `devpod-cli` binary/executable. If bundled correctly, will be placed next to the desktop app executable.
fn get_cli_path() -> Result<PathBuf, std::io::Error> {
    let mut exe_path = env::current_exe()?;
    exe_path.pop();
    exe_path.push(DEVPOD_BINARY_NAME);

    Ok(exe_path)
}

#[cfg(not(target_os = "windows"))]
fn install(_app_handle: AppHandle, force: bool) -> Result<(), InstallCLIError> {
    use anyhow::Context;
    use dirs::home_dir;
    use log::{info, warn};
    use std::{fs::remove_file, os::unix::fs::symlink};

    let cli_path = get_cli_path().map_err(InstallCLIError::NoExePath)?;

    // The binary we ship with is `devpod-cli`, but we want to link it to `devpod` so that users can just run `devpod` in their terminal
    let mut target_paths: Vec<PathBuf> = vec![];

    // /usr/local/bin/devpod
    let raw_system_bin = format!("/usr/local/bin/{}", "devpod");
    target_paths.push(PathBuf::from(&raw_system_bin));

    if force {
        info!("Attempting to force install CLI");
        let script = format!("osascript -e \"do shell script \\\"mkdir -p /usr/local/bin && ln -sf '{}' '{}'\\\" with administrator privileges\"", cli_path.to_string_lossy(), raw_system_bin);
        let status = std::process::Command::new("sh")
            .arg("-c")
            .arg(script)
            .status()
            .map_err(anyhow::Error::msg)
            .map_err(InstallCLIError::Link)?;
        info!("Status: {}", status);

        return Ok(());
    }

    if let Some(home) = home_dir() {
        // $HOME/bin/devpod
        let mut user_bin = home.clone();
        user_bin.push("bin/devpod");

        // $HOME/.local/bin/devpod
        let mut user_local_bin = home;
        user_local_bin.push(".local/bin/devpod");

        target_paths.push(user_local_bin);
        target_paths.push(user_bin);
    }

    let mut latest_error: Option<InstallCLIError> = None;
    let is_on_tmpfs = is_tmpfs(&cli_path.as_path());

    for target_path in target_paths {
        let str_target_path = target_path.to_string_lossy();
        match target_path.try_exists() {
            Ok(exists) => {
                if exists {
                    // Remove link before attempting to create another one
                    if let Err(err) = remove_file(&target_path)
                        .with_context(|| format!("path: {}", str_target_path))
                        .map_err(InstallCLIError::Link)
                    {
                        warn!(
                            "Failed to remove link: {}; Retrying with other paths...",
                            err
                        );
                        continue;
                    };
                }
            }
            _ => { /* fallthrough */ }
        }
        info!(
            "Attempting to link cli to {}",
            target_path.to_string_lossy()
        );

        let mut is_flatpak = false;

        match env::var("FLATPAK_ID") {
            Ok(_) => is_flatpak = true,
            Err(_) => is_flatpak = false,
        }

        if is_flatpak {
            match copy(cli_path.clone(), &target_path)
                .with_context(|| format!("path: {}", str_target_path))
                .map_err(InstallCLIError::Link)
            {
                Ok(..) => {
                    return Ok(());
                }
                Err(err) => {
                    warn!(
                        "Failed to copy from {} to {}: {}; Retrying with other paths...",
                        cli_path.to_string_lossy(),
                        target_path.to_string_lossy(),
                        err
                    );
                    latest_error = Some(err);
                }
            }
        } else {
            let operation = if is_on_tmpfs { copy } else { symlink };

            match operation(cli_path.clone(), &target_path)
                .with_context(|| format!("path: {}", str_target_path))
                .map_err(InstallCLIError::Link)
            {
                Ok(..) => {
                    return Ok(());
                }
                Err(err) => {
                    warn!(
                        "Failed to link to {}: {}; Retrying with other paths...",
                        target_path.to_string_lossy(),
                        err
                    );
                    latest_error = Some(err);
                }
            }
        }
    }

    if let Some(err) = latest_error {
        return Err(err);
    }

    Ok(())
}

fn copy<P: AsRef<Path>, Q: AsRef<Path>>(from: P, to: Q) -> std::io::Result<()> {
    std::fs::copy(from, to).map(|_| ())
}

#[cfg(not(target_os = "windows"))]
fn is_tmpfs(path: &Path) -> bool {
    let mountpoint_file = match std::fs::read_to_string("/proc/mounts") {
        Ok(contents) => contents,
        Err(_) => return false,
    };

    let mount_lines = mountpoint_file.lines();
    let fs = match find_fs_type(path, &mount_lines) {
        Some(contents) => contents,
        None => return false,
    };

    return fs.to_string() == "tmpfs" || fs.to_string().contains("fuse");
}

#[cfg(not(target_os = "windows"))]
fn find_fs_type(curr_path: &Path, mount_lines: &Lines) -> Option<String> {
    for line in mount_lines.clone() {
        let columns: Vec<&str> = line.split_whitespace().collect();

        if &curr_path.to_str()? == columns.get(1)? {
            return Some(columns.get(2)?.to_string());
        }
    }

    return find_fs_type(curr_path.parent()?, mount_lines);
}

#[cfg(target_os = "windows")]
fn install(app_handle: AppHandle, force: bool) -> Result<(), InstallCLIError> {
    use log::error;
    use tauri::Manager;
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
        .path()
        .app_data_dir()
        .map_err(|_|InstallCLIError::DataDir)?;
    bin_dir.push("bin");

    // Create binary directory in app dir and write bin_files to disk
    // These will be stored in a /bin folder under our control, usually `%APP_DIR%/sh.loft.devpod/bin`
    let cli_path = cli_path.to_str().ok_or(InstallCLIError::PathConversion)?;

    let sh_file = BinFile {
        name: "devpod".to_string(),
        // WARN: we actually need to debug print here because this escapes the backslash to `\\` and will then be recognised by the shell
        content: format!("#!/usr/bin/env sh\n{:?}.exe \"$@\" \nexit $?", cli_path),
    };

    let cmd_file = BinFile {
        name: format!("{}.cmd", "devpod".to_string()),
        content: format!("@echo off\n\"{}.exe\" %*", cli_path),
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
