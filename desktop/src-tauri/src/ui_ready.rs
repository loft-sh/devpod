use crate::{
    commands::DevpodCommandError, providers::ProvidersState, system_tray::SystemTray,
    workspaces::WorkspacesState, AppState,
};
use std::{
    sync::{mpsc, Arc},
    thread, time,
};
use tauri::{AppHandle, Manager};

enum Update {
    Providers(ProvidersState),
    Workspaces(WorkspacesState),
}

#[tauri::command]
pub fn ui_ready(
    app_handle: AppHandle,
    state: tauri::State<'_, AppState>,
) -> Result<(), DevpodCommandError> {
    let sleep_duration = time::Duration::from_millis(1_000);
    let (tx, rx) = mpsc::channel::<Update>();

    let workspaces_tx = tx.clone();

    thread::spawn(move || loop {
        let workspaces = WorkspacesState::load().unwrap();
        workspaces_tx.send(Update::Workspaces(workspaces)).unwrap();

        thread::sleep(sleep_duration);
    });

    let providers_state = Arc::clone(&state.providers);
    let workspaces_state = Arc::clone(&state.workspaces);
    let tray_handle = app_handle.tray_handle();

    // Handle updates from background threads.
    thread::spawn(move || {
        while let Ok(msg) = rx.recv() {
            match msg {
                Update::Providers(providers) => {
                    let current_providers = &mut *providers_state.lock().unwrap();

                    if current_providers != &providers {
                        app_handle
                            .emit_all("providers", &providers)
                            .expect("should be able to emit providers");
                        *current_providers = providers;
                    }
                }
                Update::Workspaces(workspaces) => {
                    let current_workspaces = &mut *workspaces_state.lock().unwrap();

                    if current_workspaces != &workspaces {
                        app_handle
                            .emit_all("workspaces", &workspaces)
                            .expect("should be able to emit workspaces");
                        *current_workspaces = workspaces;
                    }
                }
            }
        }
    });

    let workspaces = WorkspacesState::load().unwrap();

    let new_menu =
        SystemTray::new().build_with_submenus(vec![Box::new(&workspaces)]);
    tray_handle
        .set_menu(new_menu)
        .expect("should be able to set menu");

    Ok(())
}
