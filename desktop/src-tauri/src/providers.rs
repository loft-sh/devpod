use crate::{
    commands::{list_providers::ListProvidersCommand, DevpodCommandConfig, DevpodCommandError},
    system_tray::ToSystemTraySubmenu,
};
use chrono::DateTime;
use log::trace;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use tauri::{api::process::Command, CustomMenuItem, SystemTrayMenu, SystemTraySubmenu};

#[derive(Serialize, Deserialize, Debug)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
pub struct ProvidersState {
    default_provider: Option<String>,
    providers: Providers,
}
impl ProvidersState {
    pub const IDENTIFIER_PREFIX: &str = "providers";

    fn item_id(id: &String) -> String {
        format!("{}-{}", Self::IDENTIFIER_PREFIX, id)
    }
}
impl ProvidersState {
    pub fn load() -> Result<ProvidersState, DevpodCommandError> {
        trace!("loading providers");

        let list_providers_cmd = ListProvidersCommand::new();
        let config = list_providers_cmd.config();

        // TODO: maybe refactor into `Command` type
        let output = Command::new_sidecar(config.binary_name())
            .expect("should have found `devpod` binary")
            .args(config.args())
            .output()
            .expect("should have spawned `devpod`");

        list_providers_cmd.deserialize(&output.stdout)
    }
}

impl ToSystemTraySubmenu for ProvidersState {
    fn to_submenu(&self) -> tauri::SystemTraySubmenu {
        let mut providers_menu = SystemTrayMenu::new();
        for (provider_name, _value) in &self.providers {
            let mut item = CustomMenuItem::new(Self::item_id(provider_name), provider_name);
            if Some(provider_name.to_string()) == self.default_provider {
                item = item.selected();
            }

            providers_menu = providers_menu.add_item(item);
        }

        SystemTraySubmenu::new("Providers", providers_menu)
    }
}

type Providers = HashMap<String, Provider>;

#[derive(Serialize, Deserialize, Debug)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
struct Provider {
    options: Option<HashMap<String, ProviderOption>>,
}

#[derive(Serialize, Deserialize, Debug)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
struct ProviderOption {
    value: Option<String>,
    local: Option<bool>,
    retrieved: Option<DateTime<chrono::Utc>>,
}
