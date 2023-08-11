use crate::{
    custom_protocol::{OpenWorkspaceMsg, ParseError},
    window::WindowHelper,
    AppHandle,
};
use log::warn;
use serde::Serialize;
use std::collections::VecDeque;
use tauri::Manager;
use tokio::sync::mpsc::Receiver;

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

                    self.app_handle.get_window("main").map(|w| w.show());
                    while let Some(msg) = self.message_buffer.pop_front() {
                        let emit_result = self.app_handle.emit_all("event", msg);
                        if let Err(err) = emit_result {
                            warn!("Error sending message: {}", err);
                        }
                    }
                }
                UiMessage::ExitRequested => {
                    self.is_ready = false;
                }
                // send all other messages to the UI
                _ => self.handle_msg(ui_msg),
            }
        }
    }

    fn handle_msg(&mut self, msg: UiMessage) {
        if self.is_ready {
            self.app_handle.get_window("main").map(|w| w.show());
            let _ = self.app_handle.emit_all("event", msg);
        } else {
            // recreate window
            self.message_buffer.push_back(msg);
            let _ = self.window_helper.new_main(self.app_name.clone());
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
    OpenWorkspaceFailed(ParseError),
}

#[derive(Debug, Serialize, Clone)]
pub struct ShowToastMsg {
    title: String,
    message: String,
    status: ToastStatus,
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
