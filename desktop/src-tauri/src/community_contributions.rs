use crate::AppState;

const RAW_COMMUNITY_YAML: &str = include_str!("../../../community.yaml");

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

#[tauri::command]
pub fn get_contributions(state: tauri::State<'_, AppState>) -> Result<CommunityContributions, ()> {
    let community_contributions = state.community_contributions.lock().unwrap();

    Ok(community_contributions.clone())
}
