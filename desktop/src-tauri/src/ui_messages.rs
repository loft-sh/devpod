use crate::AppState;
use crate::{custom_protocol::ParseError, window::WindowHelper, AppHandle};
use log::{error, info, warn};
use serde::{de, Deserialize, Serialize};
use std::collections::{HashMap, VecDeque};
use tauri::{Emitter, Manager, State};
use tauri_plugin_notification::NotificationExt;
use tokio::sync::mpsc::Receiver;

pub async fn send_ui_message(
    app_state: State<'_, AppState>,
    msg: UiMessage,
    log_msg_on_failure: &str,
) {
    if let Err(err) = app_state.ui_messages.send(msg).await {
        error!("{}: {:?}, {}", log_msg_on_failure, err.0, err);
    };
}

#[derive(Debug, Clone)]
pub struct UiMessageHelper {
    app_handle: AppHandle,
    app_name: String,
    window_helper: WindowHelper,
    message_buffer: VecDeque<UiMessage>,
    is_ready: bool,
}

impl UiMessageHelper {
    pub fn new(app_handle: AppHandle, app_name: String, window_helper: WindowHelper) -> Self {
        Self {
            app_handle,
            app_name,
            window_helper,
            message_buffer: VecDeque::new(),
            is_ready: false,
        }
    }

    pub async fn listen(&mut self, mut receiver: Receiver<UiMessage>) {
        while let Some(ui_msg) = receiver.recv().await {
            match ui_msg {
                UiMessage::Ready => {
                    self.is_ready = true;

                    self.app_handle.get_webview_window("main").map(|w| w.show());
                    while let Some(msg) = self.message_buffer.pop_front() {
                        let emit_result = self.app_handle.emit("event", msg);
                        if let Err(err) = emit_result {
                            warn!("Error sending message: {}", err);
                        }
                    }
                }
                UiMessage::ExitRequested => {
                    self.is_ready = false;
                }
                UiMessage::LoginRequired(msg) => {
                    info!("Login required: {} {}", msg.host, msg.provider);

                    let main_window = self.app_handle.get_webview_window("main");
                    if !self.is_ready || main_window.is_none() {
                        // send os notification if we aren't ready to display the main window
                        let title = "Login required".to_string();
                        let body = format!(
                            "You have been logged out. Please log back in to {}",
                            msg.host,
                        );
                        let _ = self
                            .app_handle
                            .notification()
                            .builder()
                            .title(title)
                            .body(body)
                            .show();
                        continue;
                    }

                    // let main window handle
                    let _ = self.app_handle.emit("event", UiMessage::LoginRequired(msg));
                }
                // send all other messages to the UI
                _ => self.handle_msg(ui_msg),
            }
        }
    }

    fn handle_msg(&mut self, msg: UiMessage) {
        if self.is_ready {
            self.app_handle.get_webview_window("main").map(|w| w.show());
            let _ = self.app_handle.emit("event", msg);
        } else {
            // recreate window
            self.message_buffer.push_back(msg);

            // create a new main window if we can't find it
            let main_window = self.app_handle.get_webview_window("main");
            if main_window.is_none() {
                let _ = self.window_helper.new_main(self.app_name.clone());
            }
        }
    }
}

#[derive(Debug, Serialize, Clone)]
#[serde(tag = "type")]
#[allow(dead_code)]
pub enum UiMessage {
    Ready,
    ExitRequested,
    ShowDashboard,
    ShowToast(ShowToastMsg),
    OpenWorkspace(OpenWorkspaceMsg),
    OpenProInstance(OpenProInstanceMsg),
    SetupPro(SetupProMsg),
    ImportWorkspace(ImportWorkspaceMsg),
    CommandFailed(ParseError),
    LoginRequired(LoginRequiredMsg),
}

#[derive(Debug, Serialize, Clone)]
pub struct ShowToastMsg {
    title: String,
    message: String,
    status: ToastStatus,
}

impl ShowToastMsg {
    pub fn new(title: String, message: String, status: ToastStatus) -> Self {
        Self {
            title,
            message,
            status,
        }
    }
}

// WARN: Needs to match the UI's toast status
#[derive(Debug, Serialize, Clone)]
#[serde(rename_all = "lowercase")]
#[allow(dead_code)]
pub enum ToastStatus {
    Success,
    Error,
    Warning,
    Info,
    Loading,
}

#[derive(Debug, PartialEq, Serialize, Deserialize, Clone)]
#[serde(deny_unknown_fields)]
pub struct OpenWorkspaceMsg {
    #[serde(rename(deserialize = "workspace"))]
    pub workspace_id: Option<String>,
    #[serde(rename(deserialize = "provider"))]
    pub provider_id: Option<String>,
    pub ide: Option<String>,
    pub source: Option<String>,
}

#[derive(Debug, PartialEq, Serialize, Deserialize, Clone)]
#[serde(deny_unknown_fields)]
pub struct OpenProInstanceMsg {
    pub host: Option<String>,
}

#[derive(Debug, PartialEq, Serialize, Clone)]
#[serde(deny_unknown_fields)]
pub struct ImportWorkspaceMsg {
    pub workspace_id: String,
    pub workspace_uid: String,
    pub devpod_pro_host: String,
    pub project: String,
    pub options: HashMap<String, String>,
}

impl<'de> Deserialize<'de> for ImportWorkspaceMsg {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let mut options = HashMap::deserialize(deserializer)?;

        let workspace_id = options
            .remove("workspace-id")
            .ok_or_else(|| de::Error::missing_field("workspace-id"))?;

        let workspace_uid = options
            .remove("workspace-uid")
            .ok_or_else(|| de::Error::missing_field("workspace-uid"))?;

        let devpod_pro_host = options
            .remove("devpod-pro-host")
            .ok_or_else(|| de::Error::missing_field("devpod-pro-host"))?;

        let project = options
            .remove("project")
            .ok_or_else(|| de::Error::missing_field("project"))?;

        Ok(ImportWorkspaceMsg {
            workspace_id,
            workspace_uid,
            devpod_pro_host,
            project,
            options,
        })
    }
}

#[derive(Debug, PartialEq, Serialize, Clone)]
#[serde(deny_unknown_fields)]
pub struct SetupProMsg {
    pub host: String,
    #[serde(rename(serialize = "accessKey"))]
    pub access_key: Option<String>,
    pub options: Option<HashMap<String, String>>,
}

impl<'de> Deserialize<'de> for SetupProMsg {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let mut all_fields = HashMap::<String, String>::deserialize(deserializer)?;

        let host = all_fields
            .remove("host")
            .ok_or_else(|| de::Error::missing_field("host"))?;

        let mut access_key = all_fields.remove("access_key");
        if access_key.is_none() {
            access_key = all_fields.remove("accessKey");
        }

        // Options are urlencoded
        let options = all_fields.remove("options");
        if let Some(options) = options {
            let options =
                serde_urlencoded::from_str::<Vec<(String, String)>>(&options).map_err(|err| {
                    de::Error::custom(format!("Failed to url decode options: {}", err))
                })?;
            let options =
                serde_json::from_str::<HashMap<String, String>>(&options[0].0).map_err(|err| {
                    de::Error::custom(format!("Failed to json parse options: {}", err))
                })?;

            Ok(SetupProMsg {
                host,
                access_key,
                options: Some(options),
            })
        } else {
            Ok(SetupProMsg {
                host,
                access_key,
                options: None,
            })
        }
    }
}

impl OpenWorkspaceMsg {
    pub fn empty() -> OpenWorkspaceMsg {
        OpenWorkspaceMsg {
            workspace_id: None,
            provider_id: None,
            ide: None,
            source: None,
        }
    }
    pub fn with_id(id: String) -> OpenWorkspaceMsg {
        OpenWorkspaceMsg {
            workspace_id: Some(id),
            provider_id: None,
            ide: None,
            source: None,
        }
    }
}

#[derive(Debug, PartialEq, Serialize, Clone)]
#[serde(deny_unknown_fields)]
pub struct LoginRequiredMsg {
    pub host: String,
    pub provider: String,
}
