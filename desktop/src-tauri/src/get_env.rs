use crate::{commands::DevpodCommandError, AppState, UiMessage};
use log::error;

#[tauri::command]
pub fn get_env(name: &str) -> String {
    std::env::var(String::from(name)).unwrap_or(String::from(""))
}
