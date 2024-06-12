use crate::{commands::DevpodCommandError, AppState, UiMessage};
use log::{error, info, warn};
use std::path::Path;

#[tauri::command]
pub fn file_exists(filepath: &str) -> bool {
    info!("finding file in {}", filepath);
    return Path::new(&filepath).exists();
}
