use crate::{commands::DevpodCommandError, AppState, UiMessage};
use log::error;

// This command is expected to be invoked exactly once per window
#[tauri::command]
pub async fn ui_ready(state: tauri::State<'_, AppState>) -> Result<(), DevpodCommandError> {
    if let Err(err) = state.ui_messages.send(UiMessage::Ready).await {
        error!("Failed to broadcast UI ready message: {:?}", err);
    }

    Ok(())
}
