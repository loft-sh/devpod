use crate::ui_messages::{
    send_ui_message, ImportWorkspaceMsg, OpenWorkspaceMsg, SetupProMsg, ShowToastMsg, ToastStatus,
    UiMessage,
};
use crate::AppState;
use log::{error, info, warn};
use serde::{Deserialize, Serialize};
use std::env;
use std::path::Path;
use tauri::{AppHandle, Manager, State};
use thiserror::Error;
use url::Url;

// Should match the one from "tauri.config.json" and "Info.plist"
const APP_IDENTIFIER: &str = "sh.loft.devpod";
const APP_URL_SCHEME: &str = "devpod";

pub struct CustomProtocol;

pub struct Request {
    host: String,
    query: String,
}

pub struct UrlParser {}

impl UrlParser {
    const ALLOWED_METHODS: [&'static str; 3] = ["open", "import", "pro"];

    fn get_host(url: &Url) -> String {
        url.host_str().unwrap_or("no host").to_string()
    }

    fn parse_raw_url(url_scheme: &str) -> Result<Url, ParseError> {
        Url::parse(url_scheme).map_err(|_| ParseError::InvalidQuery(url_scheme.to_string()))
    }

    fn is_allowed_method(host_str: &str) -> bool {
        Self::ALLOWED_METHODS.contains(&host_str)
    }

    fn parse_query(url: &Url) -> String {
        url.query().unwrap_or("").to_string()
    }

    pub fn parse(url_scheme: &str) -> Result<Request, ParseError> {
        let url = Self::parse_raw_url(url_scheme)?;
        let host_str = Self::get_host(&url);

        if !Self::is_allowed_method(&host_str) {
            return Err(ParseError::UnsupportedHost(host_str));
        }
        return Ok(Request {
            host: host_str,
            query: Self::parse_query(&url),
        });
    }
}

pub struct OpenHandler {}

impl OpenHandler {
    pub async fn handle(msg: Result<OpenWorkspaceMsg, ParseError>, app_state: State<'_, AppState>) {
        match msg {
            Ok(msg) => Self::handle_ok(msg, app_state).await,
            Err(err) => Self::handle_error(err, app_state).await,
        }
    }

    async fn handle_ok(msg: OpenWorkspaceMsg, app_state: State<'_, AppState>) {
        // try to send to UI if ready, otherwise buffer and let ui_ready handle
        send_ui_message(
            app_state,
            UiMessage::OpenWorkspace(msg),
            "Failed to broadcast custom protocol message",
        )
        .await;
    }

    async fn handle_error(err: ParseError, app_state: State<'_, AppState>) {
        #[cfg(not(target_os = "windows"))]
        send_ui_message(
            app_state,
            UiMessage::CommandFailed(err),
            "Failed to broadcast invalid custom protocol message",
        )
        .await;
    }
}

pub struct ImportHandler {}

impl ImportHandler {
    pub async fn handle(
        msg: Result<ImportWorkspaceMsg, ParseError>,
        app_state: State<'_, AppState>,
    ) {
        match msg {
            Ok(msg) => Self::handle_ok(msg, app_state).await,
            Err(err) => Self::handle_error(err, app_state).await,
        }
    }

    async fn handle_ok(msg: ImportWorkspaceMsg, app_state: State<'_, AppState>) {
        // try to send to UI if ready, otherwise buffer and let ui_ready handle
        send_ui_message(
            app_state,
            UiMessage::ImportWorkspace(msg),
            "Failed to broadcast custom protocol message",
        )
        .await;
    }

    async fn handle_error(err: ParseError, app_state: State<'_, AppState>) {
        #[cfg(not(target_os = "windows"))]
        send_ui_message(
            app_state,
            UiMessage::CommandFailed(err),
            "Failed to broadcast invalid custom protocol message",
        )
        .await;
    }
}

pub struct ProHandler {}

impl ProHandler {
    pub async fn handle(msg: Result<SetupProMsg, ParseError>, app_state: State<'_, AppState>) {
        match msg {
            Ok(msg) => Self::handle_ok(msg, app_state).await,
            Err(err) => Self::handle_error(err, app_state).await,
        }
    }

    async fn handle_ok(msg: SetupProMsg, app_state: State<'_, AppState>) {
        // try to send to UI if ready, otherwise buffer and let ui_ready handle
        send_ui_message(
            app_state,
            UiMessage::SetupPro(msg),
            "Failed to broadcast custom protocol message",
        )
        .await;
    }

    async fn handle_error(err: ParseError, app_state: State<'_, AppState>) {
        #[cfg(not(target_os = "windows"))]
        send_ui_message(
            app_state,
            UiMessage::CommandFailed(err),
            "Failed to broadcast invalid custom protocol message",
        )
        .await;
    }
}

impl CustomProtocol {
    pub fn init() -> Self {
        tauri_plugin_deep_link::prepare(APP_IDENTIFIER);
        Self {}
    }

    pub fn forward_deep_link() {
        #[cfg(target_os = "linux")]
        {
            use std::{
                fs::{remove_file},
                io::{ErrorKind, Write},
                os::unix::net::{UnixStream}
            };

            let addr = format!("/tmp/{}-deep-link.sock", APP_IDENTIFIER);

            match UnixStream::connect(&addr) {
                Ok(mut stream) => {
                    if let Err(io_err) = stream.write_all(std::env::args().nth(1).unwrap_or_default().as_bytes())
                    {
                        log::error!(
                            "Error sending message to primary instance: {}",
                            io_err.to_string()
                        );
                    };
                }
                Err(err) => {
                    log::error!("Error creating socket listener: {}", err.to_string());
                    if err.kind() == ErrorKind::ConnectionRefused {
                        let _ = remove_file(&addr);
                    }
                }
            };
        }

        #[cfg(target_os = "windows")]
        {
            use std::io::Write;
            use interprocess::local_socket::{LocalSocketListener, LocalSocketStream};

            if let Ok(mut conn) = LocalSocketStream::connect(APP_IDENTIFIER) {
                if let Err(io_err) = conn.write_all(std::env::args().nth(1).unwrap_or_default().as_bytes())
                {
                    log::error!(
                        "Error sending message to primary instance: {}",
                        io_err.to_string()
                    );
                };
                let _ = conn.write_all(b"\n");
            }
        }

    }

    pub fn setup(&self, app: AppHandle) {
        let app_handle = app.clone();

        let result = tauri_plugin_deep_link::register(APP_URL_SCHEME, move |url_scheme| {
            tauri::async_runtime::block_on(async {
                info!("App opened with URL: {:?}", url_scheme.to_string());

                let request = UrlParser::parse(&url_scheme.to_string());
                let app_state = app_handle.state::<AppState>();
                if let Err(err) = request {
                    warn!("Failed to broadcast custom protocol message: {:?}", err);
                    return;
                }
                let request = request.unwrap();

                match request.host.as_str() {
                    "open" => {
                        let msg = CustomProtocol::parse(&request);
                        OpenHandler::handle(msg, app_state).await
                    }
                    "import" => {
                        let msg = CustomProtocol::parse(&request);
                        ImportHandler::handle(msg, app_state).await
                    }
                    "pro" => {
                        let msg = CustomProtocol::parse(&request);
                        ProHandler::handle(msg, app_state).await
                    }
                    _ => {}
                }
            })
        });

        #[cfg(target_os = "linux")]
        {
            match result {
                Ok(..) => {}
                Err(error) => {
                    let mut is_flatpak = false;

                    match env::var("FLATPAK_ID") {
                        Ok(_) => is_flatpak = true,
                        Err(_) => is_flatpak = false,
                    }

                    if !is_flatpak {
                        let msg = "Either update-desktop-database or xdg-mime are missing. Please make sure they are available on your system";
                        log::warn!("Custom protocol setup failed; {}: {}", msg, error);

                        tauri::async_runtime::block_on(async {
                            let app_state = app.state::<AppState>();
                            let show_toast_msg = ShowToastMsg::new(
                                "Custom protocol handling needs to be configured".to_string(),
                                msg.to_string(),
                                ToastStatus::Warning,
                            );
                            if let Err(err) = app_state
                                .ui_messages
                                .send(UiMessage::ShowToast(show_toast_msg))
                                .await
                            {
                                log::error!(
                                    "Failed to broadcast show toast message: {:?}, {}",
                                    err.0,
                                    err
                                );
                            };
                        })
                    }
                }
            };
        }

        let _ = result;
    }

    fn parse<'a, Msg>(request: &'a Request) -> Result<Msg, ParseError>
    where
        Msg: Deserialize<'a>,
    {
        serde_qs::from_str::<Msg>(&request.query)
            .map_err(|_| ParseError::InvalidQuery(request.query.clone()))
    }
}

#[derive(Error, Debug, Clone, Serialize)]
pub enum ParseError {
    #[error("Unsupported host: {0}")]
    UnsupportedHost(String),
    #[error("Unsupported query arguments: {0}")]
    InvalidQuery(String),
}

#[cfg(test)]
mod tests {
    mod url_parser {
        use super::super::*;

        #[test]
        fn should_parse() {
            let url_str = "devpod://open?workspace=workspace";
            let request = UrlParser::parse(&url_str).unwrap();

            assert_eq!(request.host, "open".to_string());
            assert_eq!(request.query, "workspace=workspace".to_string());
        }

        #[test]
        fn should_parse_with_empty_query() {
            let url_str = "devpod://import";
            let request = UrlParser::parse(&url_str).unwrap();

            assert_eq!(request.host, "import".to_string());
            assert_eq!(request.query, "".to_string());
        }

        #[test]
        #[should_panic]
        fn should_fail_on_invalid_method() {
            let url_str = "devpod://something";
            let _ = UrlParser::parse(&url_str).unwrap();
        }

        #[test]
        #[should_panic]
        fn should_fail_on_invalid_scheme() {
            let url_str = "invalid-scheme";
            let _ = UrlParser::parse(&url_str).unwrap();
        }
    }

    mod custom_handler_open {
        use crate::custom_protocol::OpenWorkspaceMsg;

        use super::super::*;

        #[test]
        fn should_parse_full() {
            let url_str =
                "devpod://open?workspace=workspace&provider=provider&source=https://github.com/test123&ide=vscode";
            let request = UrlParser::parse(&url_str).unwrap();
            let got: OpenWorkspaceMsg = CustomProtocol::parse(&request).unwrap();

            assert_eq!(got.workspace_id, Some("workspace".to_string()));
            assert_eq!(got.provider_id, Some("provider".into()));
            assert_eq!(got.source, Some("https://github.com/test123".to_string()));
            assert_eq!(got.ide, Some("vscode".into()));
        }

        #[test]
        fn should_parse_workspace() {
            let url_str = "devpod://open?workspace=some-workspace";
            let request = UrlParser::parse(&url_str).unwrap();
            let got: OpenWorkspaceMsg = CustomProtocol::parse(&request).unwrap();

            assert_eq!(got.workspace_id, Some("some-workspace".to_string()));
            assert_eq!(got.provider_id, None);
            assert_eq!(got.source, None);
            assert_eq!(got.ide, None)
        }

        #[test]
        fn should_parse() {
            let url_str = "devpod://open?source=some-source";
            let request = UrlParser::parse(&url_str).unwrap();
            let got: OpenWorkspaceMsg = CustomProtocol::parse(&request).unwrap();

            assert_eq!(got.workspace_id, None);
            assert_eq!(got.provider_id, None);
            assert_eq!(got.source, Some("some-source".to_string()));
            assert_eq!(got.ide, None)
        }
    }

    mod custom_handler_import {
        use crate::custom_protocol::ImportWorkspaceMsg;

        use super::super::*;

        #[test]
        fn should_parse_full() {
            let url_str =
                "devpod://import?workspace-id=workspace&workspace-uid=uid&devpod-pro-host=devpod.pro&other=other&project=foo";
            let request = UrlParser::parse(&url_str).unwrap();

            let got: ImportWorkspaceMsg = CustomProtocol::parse(&request).unwrap();

            assert_eq!(got.workspace_id, "workspace".to_string());
            assert_eq!(got.workspace_uid, "uid".to_string());
            assert_eq!(got.project, "foo".to_string());
            assert_eq!(got.devpod_pro_host, "devpod.pro".to_string());
            assert_eq!(got.options.get("other"), Some(&"other".to_string()));
        }

        #[test]
        #[should_panic]
        fn should_fail_on_missing_workspace_id() {
            let url_str =
                "devpod://import?workspace-uid=uid&devpod-pro-host=devpod.pro&other=other";
            let request = UrlParser::parse(&url_str).unwrap();

            let got: Result<ImportWorkspaceMsg, ParseError> = CustomProtocol::parse(&request);
            got.unwrap();
        }
    }

    mod custom_handler_pro_setup {
        use crate::custom_protocol::SetupProMsg;

        use super::super::*;

        #[test]
        fn should_parse_full() {
            let url_str = "devpod://pro/setup?host=foo&access_key=bar";
            let request = UrlParser::parse(&url_str).unwrap();

            let got: SetupProMsg = CustomProtocol::parse(&request).unwrap();

            assert_eq!(got.host, "foo".to_string());
            assert_eq!(got.access_key, Option::Some("bar".to_string()));
        }

        #[test]
        fn should_parse_host() {
            let url_str = "devpod://pro/setup?host=localhost%3A8080";
            let request = UrlParser::parse(&url_str).unwrap();

            let got: SetupProMsg = CustomProtocol::parse(&request).unwrap();

            assert_eq!(got.host, "localhost:8080".to_string());
        }

        #[test]
        #[should_panic]
        fn should_fail_on_missing_workspace_id() {
            let url_str =
                "devpod://import?workspace-uid=uid&devpod-pro-host=devpod.pro&other=other";
            let request = UrlParser::parse(&url_str).unwrap();

            let got: Result<ImportWorkspaceMsg, ParseError> = CustomProtocol::parse(&request);
            got.unwrap();
        }
    }
}
