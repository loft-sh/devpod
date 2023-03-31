use anyhow::Context;
use std::{
    fs::{self, OpenOptions},
    io::Write,
    path::PathBuf,
};
use tauri::Wry;
use thiserror::Error;

const ACTION_LOGS_DIR: &str = "action_logs";

type AppHandle = tauri::AppHandle<Wry>;

#[derive(Error, Debug)]
pub enum ActionLogError {
    #[error("unable to get app data dir")]
    NoDir,
    #[error("unable to open file")]
    FileOpen(#[source] std::io::Error),
    #[error("unable to write to file")]
    Write(#[source] std::io::Error),
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
        .map_err(|e| ActionLogError::FileOpen(e))?;

    file.write_all(format!("{}\n", data).as_bytes())
        .map_err(|e| ActionLogError::Write(e))?;

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
        .map_err(|e| ActionLogError::FileOpen(e))?
        .lines()
        .map(|s| s.to_string())
        .collect();

    Ok(lines)
}

pub fn setup(app_handle: &AppHandle) -> anyhow::Result<()> {
    let dir_path = get_actions_dir(app_handle)?;
    let _ = fs::create_dir(&dir_path); // Make sure we have the action logs dir

    // TODO:  trim down logs to keep in sync with UI

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
