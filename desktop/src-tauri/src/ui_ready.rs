use crate::{
    commands::DevpodCommandError, system_tray::SystemTray, workspaces::WorkspacesState, AppState,
    UiMessage,
};
use std::{
    sync::{mpsc, Arc},
    thread, time,
};
use tauri::AppHandle;
use tokio::sync::OnceCell;

static INIT: OnceCell<()> = OnceCell::const_new();

enum Update {
    Workspaces(WorkspacesState),
}

// TODO: handle multiple windows
// This command is expected to be invoked exactly once per window
#[tauri::command]
pub async fn ui_ready(
    app_handle: AppHandle,
    state: tauri::State<'_, AppState>,
) -> Result<(), DevpodCommandError> {
    // Make sure we only initialize the background threads when the first ui is ready.
    INIT.get_or_init(|| async {
        let sleep_duration = time::Duration::from_millis(1_000);
        let (tx, rx) = mpsc::channel::<Update>();

        let workspaces_tx = tx.clone();

        thread::spawn(move || loop {
            let workspaces = WorkspacesState::load().unwrap();
            workspaces_tx.send(Update::Workspaces(workspaces)).unwrap();

            thread::sleep(sleep_duration);
        });

        let workspaces_state = Arc::clone(&state.workspaces);
        let tray_handle = app_handle.tray_handle();

        // Handle updates from background threads.
        thread::spawn(move || {
            while let Ok(msg) = rx.recv() {
                match msg {
                    Update::Workspaces(workspaces) => {
                        let current_workspaces = &mut *workspaces_state.lock().unwrap();

                        if current_workspaces != &workspaces {
                            *current_workspaces = workspaces;

                            // rebuild menu
                            let new_menu = SystemTray::new()
                                .build_with_submenus(vec![Box::new(current_workspaces)]);
                            tray_handle
                                .set_menu(new_menu)
                                .expect("should be able to set menu");
                        }
                    }
                }
            }
        });
    })
    .await;

    let _ = state.ui_messages.send(UiMessage::Ready);

    Ok(())
}
