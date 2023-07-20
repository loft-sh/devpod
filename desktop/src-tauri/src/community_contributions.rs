use crate::AppState;
use std::sync::Arc;

const RAW_COMMUNITY_YAML: &str = include_str!("../../../community.yaml");
const REMOTE_COMMUNITY_YAML_URL: &str =
    "https://raw.githubusercontent.com/loft-sh/devpod/main/community.yaml";

#[derive(Debug, PartialEq, Clone, serde::Serialize, serde::Deserialize)]
pub struct CommunityContributions {
    providers: Vec<CommunityProvider>,
}

#[derive(Debug, PartialEq, Clone, serde::Serialize, serde::Deserialize)]
struct CommunityProvider {
    repository: String,
}

pub fn init() -> anyhow::Result<CommunityContributions> {
    serde_yaml::from_str(RAW_COMMUNITY_YAML).map_err(|e| anyhow::anyhow!(e))
}

// update community contributions state in background by fetching latest file from github
pub fn setup(state: tauri::State<'_, AppState>) {
    let community_contributions_state = Arc::clone(&state.community_contributions);

    tauri::async_runtime::spawn(async move {
        // TODO: uncommenct once we settled on a remote strategy
        // match fetch_community_contributions().await {
        //     Ok(remote_community_contributions) => {
        //         let _ = remote_community_contributions;
        //         let mut community_contributions = community_contributions_state.lock().unwrap();
        //         *community_contributions = remote_community_contributions;
        //     }
        //     Err(e) => {
        //         warn!("Error fetching community contributions: {:?}", e);
        //     }
        // }
    });
}

async fn fetch_community_contributions() -> anyhow::Result<CommunityContributions> {
    let body = reqwest::get(REMOTE_COMMUNITY_YAML_URL)
        .await?
        .text()
        .await?;

    serde_yaml::from_str(&body).map_err(|e| anyhow::anyhow!(e))
}

#[tauri::command]
pub fn get_contributions(state: tauri::State<'_, AppState>) -> Result<CommunityContributions, ()> {
    let community_contributions = state.community_contributions.lock().unwrap();

    Ok(community_contributions.clone())
}
