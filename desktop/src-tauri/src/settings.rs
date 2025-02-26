#![allow(dead_code)]

use crate::AppHandle;
use log::error;
use serde::Serialize;
use tauri_plugin_store::StoreExt;
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
    #[serde(rename = "experimental_codium")]
    experimental_codium: bool,
    #[serde(rename = "experimental_zed")]
    experimental_zed: bool,
    #[serde(rename = "experimental_positron")]
    experimental_positron: bool,
    #[serde(rename = "experimental_rstudio")]
    experimental_rstudio: bool,
    #[serde(rename = "experimental_devPodPro")]
    experimental_devpod_pro: bool,
    #[serde(rename = "experimental_colorMode")]
    experimental_color_mode: ColorMode,
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
        // check something in auto updates
        let store = app_handle.store(SETTINGS_FILE_NAME);
        if store.is_err() {
            error!("unable to open store {}", SETTINGS_FILE_NAME);
            return false;
        }

        store
            .unwrap()
            .get("autoUpdate")
            .and_then(|v| v.as_bool())
            .unwrap_or(true)
    }
}
