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
    #[serde(rename = "experimental_multiDevcontainer")]
    experimental_multi_devcontainer: bool,
    #[serde(rename = "experimental_fleet")]
    experimental_fleet: bool,
    #[serde(rename = "experimental_jupyterNotebooks")]
    experimental_jupyter_notebooks: bool,
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
