use std::time::{Duration, Instant};

use tauri::{AppHandle, Manager, Wry};
use tauri_plugin_store::{with_store, Store, StoreCollection};

/// `measure`  the duration it took a function to execute.
#[allow(dead_code)]
pub fn measure<F>(f: F) -> Duration
where
    F: Fn(),
{
    let start = Instant::now();
    f();

    start.elapsed()
}

pub fn with_data_store<T, F: FnOnce(&mut Store<Wry>) -> Result<T, tauri_plugin_store::Error>>(
    app_handle: &AppHandle,
    filename: &str,
    f: F,
) -> anyhow::Result<T> {
    let stores = app_handle.state::<StoreCollection<Wry>>();
    let path = app_handle.path_resolver().app_data_dir();
    if path.is_none() {
        return Err(anyhow::anyhow!("AppDataDir should exist"));
    }

    let mut path = path.expect("AppDataDir should exist");
    path.push(filename);

    return with_store(app_handle.clone(), stores, path, f).map_err(anyhow::Error::from);
}
