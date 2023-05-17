use crate::AppHandle;
use anyhow::Context;
use log::info;
use std::{
    fs::{self, OpenOptions},
    io::Write,
    path::PathBuf,
    time::{Duration, SystemTime},
};
use thiserror::Error;

const ACTION_LOGS_DIR: &str = "action_logs";
const ONE_DAY: Duration = Duration::new(60 * 60 * 24, 0);

#[derive(Error, Debug)]
pub enum ActionLogError {
    #[error("unable to get app data dir")]
    NoDir,
    #[error("unable to open file")]
    FileOpen(#[source] std::io::Error),
    #[error("unable to write to file")]
    Write(#[source] std::io::Error),
    #[error("unable to delete to file")]
    FileDelete(#[source] std::io::Error),
}
impl serde::Serialize for ActionLogError {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}

#[tauri::command]
pub fn write_action_log(
    app_handle: AppHandle,
    action_id: String,
    data: String,
) -> Result<(), ActionLogError> {
    let mut path = get_actions_dir(&app_handle).map_err(|_| ActionLogError::NoDir)?;
    path.push(format!("{}.log", &action_id));

    let mut file = OpenOptions::new()
        .create(true)
        .append(true)
        .open(path)
        .map_err(ActionLogError::FileOpen)?;

    file.write_all(format!("{}\n", data).as_bytes())
        .map_err(ActionLogError::Write)?;

    Ok(())
}

#[tauri::command]
pub fn get_action_logs(
    app_handle: AppHandle,
    action_id: String,
) -> Result<Vec<String>, ActionLogError> {
    let mut path = get_actions_dir(&app_handle).map_err(|_| ActionLogError::NoDir)?;
    path.push(format!("{}.log", &action_id));

    let lines = fs::read_to_string(path)
        .map_err(ActionLogError::FileOpen)?
        .lines()
        .map(|s| s.to_string())
        .collect();

    Ok(lines)
}

#[tauri::command]
pub fn sync_action_logs(app_handle: AppHandle, actions: Vec<String>) -> Result<(), ActionLogError> {
    let now = SystemTime::now();
    let dir_path = get_actions_dir(&app_handle).map_err(|_| ActionLogError::NoDir)?;
    let paths_to_delete = fs::read_dir(dir_path)
        .map_err(ActionLogError::FileOpen)?
        .filter_map(|r| {
            let entry = r.ok()?;
            let path = entry.path();
            let file_stem = path.file_stem()?.to_str()?;
            if actions.contains(&file_stem.to_string()) {
                return None;
            }

            let metadata = entry.metadata().ok()?;
            let created = metadata.created().ok()?;

            let elapsed = now.duration_since(created).ok()?;
            // older than a day
            if elapsed < ONE_DAY {
                return None;
            }

            Some(path)
        });

    for path in paths_to_delete {
        info!("Deleting {:?}", path);
        fs::remove_file(path).map_err(ActionLogError::FileDelete)?;
    }

    Ok(())
}

pub fn setup(app_handle: &AppHandle) -> anyhow::Result<()> {
    let dir_path = get_actions_dir(app_handle)?;
    let _ = fs::create_dir_all(dir_path); // Make sure we have the action logs dir

    Ok(())
}

fn get_actions_dir(app_handle: &AppHandle) -> anyhow::Result<PathBuf> {
    let mut dir_path = app_handle
        .path_resolver()
        .app_data_dir()
        .context("App data dir not found")?;
    dir_path.push(ACTION_LOGS_DIR);

    Ok(dir_path)
}
