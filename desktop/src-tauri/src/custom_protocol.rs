use anyhow::{Context, Result};
use log::{error, info};
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager};
use tauri_plugin_deep_link;
use thiserror::Error;
use url::Url;

use crate::{AppState, UiMessage};

// Should match the one from "tauri.config.json" and "Info.plist"
const APP_IDENTIFIER: &str = "sh.loft.devpod-desktop";
const APP_URL_SCHEME: &str = "devpod";

pub struct CustomProtocol;

#[derive(Debug, PartialEq, Serialize, Deserialize, Clone)]
pub struct OpenWorkspaceMsg {
    #[serde(rename(deserialize = "workspace"))]
    workspace_id: String,
    #[serde(rename(deserialize = "provider"))]
    provider_id: Option<String>,
    ide: Option<String>,
    source: Option<String>,
}

#[derive(Error, Debug)]
enum ParseError {
    #[error("Unsupported host: {0}")]
    UnsupportedHost(String),
}

impl CustomProtocol {
    pub fn init() -> Self {
        tauri_plugin_deep_link::prepare(APP_IDENTIFIER);
        Self {}
    }

    pub fn setup(&self, app: AppHandle) {
        tauri_plugin_deep_link::register(APP_URL_SCHEME, move |url_scheme| {
            tauri::async_runtime::block_on(async {
                info!("App opened with URL: {:?}", url_scheme.to_string());

                let msg = CustomProtocol::parse(&url_scheme.to_string());

                match msg {
                    Ok(msg) => {
                        let app_state = app.state::<AppState>();
                        // try to send to UI if ready, otherwise buffer and let ui_ready handle
                        if let Err(err) = app_state
                            .ui_messages
                            .send(UiMessage::OpenWorkspace(msg))
                            .await
                        {
                            error!(
                                "Failed to broadcast custom protocol message: {:?}, {}",
                                err.0, err
                            );
                        };
                    }
                    Err(err) => {
                        error!(
                            "Failed to parse custom protocol: {:?}, {}",
                            url_scheme.to_string(),
                            err
                        );
                    }
                }
            })
        })
        .expect("should be able to listen to custom protocols");
    }

    fn parse(url_scheme: &str) -> Result<OpenWorkspaceMsg> {
        let url = Url::parse(url_scheme)?;
        let host_str = url.host_str().unwrap_or("no host").to_string();
        if host_str != "open" {
            return Err(ParseError::UnsupportedHost(host_str).into());
        }

        serde_qs::from_str::<OpenWorkspaceMsg>(url.query().unwrap_or(""))
            .context("Failed to parse query string")
    }
}

#[cfg(test)]
mod tests {
    #[test]
    fn should_parse_full() {
        let url_str =
            "devpod://open?workspace=workspace&provider=provider&source=https://github.com/test123&ide=vscode";
        let got = super::CustomProtocol::parse(url_str).unwrap();

        assert_eq!(got.workspace_id, "workspace".to_string());
        assert_eq!(got.provider_id, Some("provider".into()));
        assert_eq!(got.source, Some("https://github.com/test123".into()));
        assert_eq!(got.ide, Some("vscode".into()));
    }

    #[test]
    fn should_parse() {
        let url_str = "devpod://open?workspace=workspace";
        let got = super::CustomProtocol::parse(url_str).unwrap();

        assert_eq!(got.workspace_id, "workspace".to_string());
        assert_eq!(got.provider_id, None);
        assert_eq!(got.source, None);
        assert_eq!(got.ide, None)
    }

    #[test]
    #[should_panic]
    fn unsupported_host() {
        let url_str = "devpod://something?workspace=workspace";
        let _ = super::CustomProtocol::parse(url_str).unwrap();
    }

    #[test]
    #[should_panic]
    fn missing_workspace_id() {
        let url_str = "devpod://open?provider=provider";
        let _ = super::CustomProtocol::parse(url_str).unwrap();
    }
}
