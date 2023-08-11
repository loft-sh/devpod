use std::collections::HashMap;

use crate::{settings::Settings, window::WindowHelper, AppHandle, AppState};
use anyhow::Context;
use chrono::{DateTime, Utc};
use lazy_static::lazy_static;
use log::{debug, error, warn};
use regex::Regex;
use reqwest::{Client, Method};
use serde::{Deserialize, Serialize};
use tauri::{api::notification::Notification, Manager};
use thiserror::Error;
use tokio::fs::File;
use ts_rs::TS;

const UPDATE_POLL_INTERVAL: std::time::Duration = std::time::Duration::from_secs(60 * 10);
const RELEASES_URL: &str = "https://api.github.com/repos/loft-sh/devpod/releases";

#[derive(Error, Debug)]
pub enum UpdateError {
    #[error("unable to get latest release {0}")]
    NoReleaseFound(String),
    #[error("failed to check for updates {0}")]
    CheckUpdate(#[from] tauri::updater::Error),
    #[error("failed to fetch releases {0}")]
    FetchRelease(#[from] anyhow::Error),
}
impl serde::Serialize for UpdateError {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}

pub type Releases = Vec<Release>;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, TS)]
#[serde(rename_all = "snake_case")]
#[ts(export)]
pub struct Release {
    #[ts(type = "String")]
    pub url: String,
    pub html_url: String,
    pub assets_url: String,
    pub upload_url: String,
    pub tarball_url: Option<String>,
    pub zipball_url: Option<String>,
    pub id: u64,
    pub node_id: String,
    pub tag_name: String,
    pub target_commitish: String,
    pub name: Option<String>,
    pub body: Option<String>,
    pub draft: bool,
    pub prerelease: bool,
    pub created_at: Option<DateTime<Utc>>,
    pub published_at: Option<DateTime<Utc>>,
    pub author: Author,
    pub assets: Vec<Asset>,
}
impl Release {
    pub fn is_pre(&self) -> bool {
        lazy_static! {
            static ref PRE_REGEX: Regex = Regex::new(r"^.*-(alpha|beta).\d*$").unwrap();
        }
        PRE_REGEX.is_match(&self.tag_name)
    }

    pub fn trim_pre(&self) -> String {
        lazy_static! {
            static ref PRE_REPL_REGEX: Regex = Regex::new(r"-(alpha|beta).\d*").unwrap();
        }
        PRE_REPL_REGEX.replace_all(&self.tag_name, "").to_string()
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, TS)]
#[serde(rename_all = "snake_case")]
#[ts(export)]
#[non_exhaustive]
pub struct Asset {
    pub url: String,
    pub browser_download_url: String,
    pub id: u64,
    pub node_id: String,
    pub name: String,
    pub label: Option<String>,
    pub state: String,
    pub content_type: String,
    pub size: i64,
    pub download_count: i64,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Hash, Eq, PartialEq, Serialize, Deserialize, TS)]
#[serde(rename_all = "snake_case")]
#[ts(export)]
#[non_exhaustive]
pub struct Author {
    pub login: String,
    pub id: u64,
    pub node_id: String,
    #[ts(type = "String")]
    pub avatar_url: String,
    pub gravatar_id: String,
    pub url: String,
    pub html_url: String,
    pub followers_url: String,
    pub following_url: String,
    pub gists_url: String,
    pub starred_url: String,
    pub subscriptions_url: String,
    pub organizations_url: String,
    pub repos_url: String,
    pub events_url: String,
    pub received_events_url: String,
    pub r#type: String,
    pub site_admin: bool,
}

#[tauri::command]
pub async fn get_releases(state: tauri::State<'_, AppState>) -> Result<Releases, ()> {
    let releases = state.releases.lock().unwrap();

    Ok(releases.to_vec())
}

#[tauri::command]
pub async fn get_pending_update(state: tauri::State<'_, AppState>) -> Result<Release, ()> {
    let release = state.pending_update.lock().unwrap();
    release.clone().ok_or(())
}

#[tauri::command]
pub async fn check_updates(app_handle: AppHandle) -> Result<bool, UpdateError> {
    match tauri::updater::builder(app_handle).check().await {
        Ok(update) => {
            let update_available = update.is_update_available();
            debug!(
                "Update check completed, update available: {}",
                update_available
            );

            return Ok(update_available);
        }
        Err(e) => {
            error!("Failed to get update: {}", e);

            return Err(UpdateError::CheckUpdate(e));
        }
    }
}

#[derive(Clone, Debug)]
pub struct UpdateHelper<'a> {
    app_handle: &'a AppHandle,
}

impl<'a> UpdateHelper<'a> {
    pub fn new(app_handle: &'a AppHandle) -> Self {
        Self {
            app_handle: &app_handle,
        }
    }

    pub async fn poll(&self) {
        {
            loop {
                // check if we have updated the app recently
                // if so, show changelog in app

                let app_handle = self.app_handle.clone();
                if let Ok(update) = tauri::updater::builder(app_handle).check().await {
                    if update.is_update_available() {
                        let new_version = update.latest_version();
                        let update_helper = UpdateHelper::new(&self.app_handle);
                        if let Err(e) = update_helper.update_app_releases(new_version).await {
                            error!("Failed to update app releases: {}", e);
                        }
                        if Settings::auto_update_enabled(&self.app_handle) {
                            if let Err(err) = update.download_and_install().await {
                                error!("Failed to download and install update: {}", err);
                            }
                        }
                    }
                }
                tokio::time::sleep(UPDATE_POLL_INTERVAL).await;
            }
        }
    }

    pub async fn update_app_releases(&self, new_version: &str) -> Result<Release, UpdateError> {
        let releases = self
            .fetch_releases()
            .await
            .map_err(UpdateError::FetchRelease)?;
        let state = self.app_handle.state::<AppState>();
        let mut releases_state = state.releases.lock().unwrap();
        *releases_state = releases;

        Ok(releases_state
            .iter()
            .find(|r| r.tag_name.contains(new_version))
            .ok_or(UpdateError::NoReleaseFound(
                "No releases found in releases state".to_string(),
            ))?
            .clone())
    }

    pub async fn fetch_releases(&self) -> anyhow::Result<Releases> {
        let per_page = 50;
        let page = 1;

        let client = Client::builder().user_agent("loft-sh/devpod").build()?;
        let request = client
            .request(Method::GET, RELEASES_URL)
            .query(&[("per_page", per_page), ("page", page)])
            .header("Accept", "application/vnd.github+json")
            .header("X-GitHub-Api-Version", "2022-11-28");

        let releases = request
            .send()
            .await?
            .json::<Vec<Release>>()
            .await
            .with_context(|| format!("Fetch releases from {}", RELEASES_URL))?;

        let releases = &releases
            .into_iter()
            .filter(|release| !release.draft)
            .map(|release| (release.tag_name.clone(), release))
            .collect::<HashMap<String, Release>>();

        let mut releases = releases
            .into_iter()
            .filter_map(|(_, release)| {
                if release.prerelease || release.is_pre() {
                    let stable_tag_name = release.trim_pre();
                    if releases.get(&stable_tag_name).is_some() {
                        return None;
                    }
                }

                Some(release.clone())
            })
            .collect::<Vec<Release>>();
        releases.sort_by(|a, b| b.tag_name.cmp(&a.tag_name));

        Ok(releases)
    }

    pub async fn handle_event(&self, updater_event: tauri::UpdaterEvent, app_identifier: &str) {
        match updater_event {
            tauri::UpdaterEvent::UpdateAvailable { version, .. } => {
                let state = self.app_handle.state::<AppState>();
                let update_installed_state = *state.update_installed.lock().unwrap();

                if Settings::auto_update_enabled(self.app_handle) || update_installed_state {
                    return;
                }

                match self.update_app_releases(&version).await {
                    Ok(release) => {
                        if let Err(err) =
                            self.notify_update_available(&release, app_identifier).await
                        {
                            warn!("Failed to send update notification: {}", err);
                        }

                        // display update available in the UI
                        let state = self.app_handle.state::<AppState>();
                        let mut pending_update_state = state.pending_update.lock().unwrap();
                        *pending_update_state = Some(release);
                    }
                    Err(e) => {
                        error!("Failed to update app releases: {}", e);
                    }
                }
            }
            // Emitted when the download is about to be started.
            tauri::UpdaterEvent::Pending => {
                debug!("update is pending");
            }
            tauri::UpdaterEvent::DownloadProgress { .. } => {}
            // Emtted when the download has finished and the update is about to be installed.
            tauri::UpdaterEvent::Downloaded => {
                debug!("update has been downloaded");
            }
            // Emitted when the update was installed. You can then ask to restart the app.
            tauri::UpdaterEvent::Updated => {
                let window_helper = WindowHelper::new(self.app_handle.clone());
                let _ = window_helper.new_update_ready_window();

                let state = self.app_handle.state::<AppState>();
                let mut pending_update_state = state.pending_update.lock().unwrap();
                *pending_update_state = None;

                let mut update_installed_state = state.update_installed.lock().unwrap();
                *update_installed_state = true;

                debug!("update has been installed");
            }
            // Emitted when the app already has the latest version installed and an update is not needed.
            tauri::UpdaterEvent::AlreadyUpToDate => {
                debug!("app is already up to date");
            }
            // Emitted when there is an error with the updater. We suggest to listen to this event even if the default dialog is enabled.
            tauri::UpdaterEvent::Error(error) => {
                error!("failed to update: {}", error);
            }
        }
    }
}

impl UpdateHelper<'_> {
    fn build_update_notification(&self, release: &Release, app_identifier: &str) -> Notification {
        Notification::new(app_identifier)
            .title("Update available")
            .body(&format!("Version {} is available", release.tag_name))
    }

    async fn notify_update_available(
        &self,
        release: &Release,
        app_identifier: &str,
    ) -> anyhow::Result<()> {
        if let Some(mut target) = self.app_handle.path_resolver().app_cache_dir() {
            target.push(format!("update_{}", release.tag_name.clone()));

            if target.exists() {
                return Ok(());
            }
            let _ = File::create(target).await?;
        }

        let _ = self
            .build_update_notification(release, app_identifier)
            .show()?;

        Ok(())
    }
}
