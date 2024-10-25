#![allow(dead_code)]

use crate::{util::with_data_store, AppHandle};
use serde::Serialize;
use ts_rs::TS;

const SETTINGS_FILE_NAME: &str = ".settings.json";

#[derive(Debug, Serialize, TS)]
#[ts(rename_all = "camelCase")]
#[ts(export)]
pub struct Settings {
    sidebar_position: SidebarPosition,
    debug_flag: bool,
    party_parrot: bool,
    #[serde(rename = "fixedIDE")]
    fixed_ide: bool,
    zoom: Zoom,
    transparency: bool,
    auto_update: bool,
    additional_cli_flags: String,
    additional_env_vars: String,
    dotfiles_url: String,
    ssh_key_path: String,
    http_proxy_url: String,
    https_proxy_url: String,
    no_proxy: String,

    // Experimental settings
    #[serde(rename = "experimental_colorMode")]
    experimental_color_mode: ColorMode,
    #[serde(rename = "experimental_multiDevcontainer")]
    experimental_multi_devcontainer: bool,
    #[serde(rename = "experimental_fleet")]
    experimental_fleet: bool,
    #[serde(rename = "experimental_jupyterNotebooks")]
    experimental_jupyter_notebooks: bool,
    #[serde(rename = "experimental_vscodeInsiders")]
    experimental_vscode_insiders: bool,
    #[serde(rename = "experimental_cursor")]
    experimental_cursor: bool,
    #[serde(rename = "experimental_positron")]
    experimental_positron: bool,
    #[serde(rename = "experimental_devPodPro")]
    experimental_devpod_pro: bool,
}

#[derive(Debug, Serialize, TS)]
#[ts(rename_all = "camelCase")]
#[ts(export)]
enum SidebarPosition {
    Left,
    Right,
}

#[derive(Debug, Serialize, TS)]
#[ts(rename_all = "camelCase")]
#[ts(export)]
enum ColorMode {
    Dark,
    Light,
}

#[derive(Debug, Serialize, TS)]
#[ts(rename_all = "camelCase")]
#[ts(export)]
enum Zoom {
    Sm,
    Md,
    Lg,
    Xl,
}

impl Settings {
    pub fn auto_update_enabled(app_handle: &AppHandle) -> bool {
        let mut is_enabled = false;
        let _ = with_data_store(&app_handle, SETTINGS_FILE_NAME, |store| {
            is_enabled = store
                .get("autoUpdate")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);

            Ok(())
        });

        return is_enabled;
    }
}
