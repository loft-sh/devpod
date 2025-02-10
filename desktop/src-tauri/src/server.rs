use crate::{ui_messages, AppHandle, AppState};
use axum::{
    extract::{
        connect_info::ConnectInfo,
        ws::{Message, WebSocket, WebSocketUpgrade},
        State as AxumState,
    },
    http::{HeaderMap, StatusCode},
    response::{IntoResponse, Response},
    routing::{get, post},
    Json, Router,
};
use http::Method;
use log::{error, info, warn};
use serde::{Deserialize, Serialize};
use std::net::SocketAddr;
use tauri::{Manager, State};
use tower_http::cors::{Any, CorsLayer};

#[derive(Clone)]
struct ServerState {
    app_handle: AppHandle,
}

pub async fn setup(app_handle: &AppHandle) -> anyhow::Result<()> {
    let state = ServerState {
        app_handle: app_handle.clone(),
    };

    let cors = CorsLayer::new()
        .allow_methods([Method::GET, Method::POST])
        .allow_headers(Any)
        .allow_origin(Any);

    let router = Router::new()
        .route("/ws", get(ws_handler))
        .route("/releases", get(releases_handler))
        .route("/child-process/signal", post(signal_handler))
        .with_state(state)
        .layer(cors);

    let listener = tokio::net::TcpListener::bind("127.0.0.1:25842").await?;
    info!("Listening on {}", listener.local_addr()?);
    return axum::serve(
        listener,
        router.into_make_service_with_connect_info::<SocketAddr>(),
    )
    .await
    .map_err(anyhow::Error::from);
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct SendSignalMessage {
    process_id: i32,
    signal: i32, // should match nix::sys::signal::Signal
}

async fn signal_handler(
    AxumState(server): AxumState<ServerState>,
    Json(payload): Json<SendSignalMessage>,
) -> impl IntoResponse {
    info!("received request to send signal {} to process {}", payload.signal, payload.process_id.to_string());
    #[cfg(not(windows))]
    {
        use nix::sys::signal::{self, kill, Signal};
        use nix::unistd::Pid;
        let pid = Pid::from_raw(payload.process_id);
        // TODO: convert payload.signal into signal
        let signal = Signal::SIGINT;
        if let Err(err) = signal::kill(pid, signal) {
            error!("Failed to kill process: {}", err);
            return StatusCode::INTERNAL_SERVER_ERROR;
        }
    }
    #[cfg(windows)]
    {
        use windows::Win32::System::Threading::{OpenProcess, TerminateProcess, PROCESS_TERMINATE};
        use windows::Win32::Foundation::{HANDLE, CloseHandle};
        use crate::util::kill_child_processes;

        kill_child_processes(payload.process_id as u32);

        unsafe {
            let handle: windows::core::Result<HANDLE> = OpenProcess(PROCESS_TERMINATE, false, payload.process_id.try_into().unwrap());
            if handle.is_err() {
                error!("unable to open process {}: {:?}", payload.process_id, handle.unwrap_err());
                return StatusCode::INTERNAL_SERVER_ERROR;
            }
            let handle: HANDLE = handle.unwrap();

            let result = TerminateProcess(handle, 1);
            CloseHandle(handle);
            if !result.as_bool() {
                error!("unable to terminate process {}", payload.process_id);
                return StatusCode::INTERNAL_SERVER_ERROR;
            }
        }
    }

    info!("successfully killed process");

    return StatusCode::OK;
}

async fn releases_handler(AxumState(server): AxumState<ServerState>) -> impl IntoResponse {
    let state = server.app_handle.state::<AppState>();
    let releases = state.releases.lock().unwrap();
    let releases = releases.clone();

    Json(releases)
}

async fn ws_handler(
    ws: WebSocketUpgrade,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    AxumState(server): AxumState<ServerState>,
) -> Response {
    let app_handle = server.app_handle;
    let user_agent = if let Some(user_agent) = headers.get("user-agent") {
        user_agent.to_str().unwrap_or("Unknown browser")
    } else {
        "Unknown browser"
    };

    info!("`{user_agent}` at {addr} connected.");
    ws.on_upgrade(move |socket| handle_socket(socket, addr, app_handle))
}

async fn handle_socket(mut socket: WebSocket, who: SocketAddr, app_handle: AppHandle) {
    while let Some(msg) = socket.recv().await {
        if let Ok(msg) = msg {
            match msg {
                Message::Text(raw_text) => 'text: {
                    info!("Received message: {}", raw_text);
                    let json = serde_json::from_str::<ui_messages::SetupProMsg>(raw_text.as_str());
                    if let Err(err) = json {
                        warn!("Failed to parse json: {}", err);
                        // drop message
                        break 'text;
                    };

                    let payload = json.unwrap(); // we can safely unwrap here, checked for error earlier
                    ui_messages::send_ui_message(
                        app_handle.state::<AppState>(),
                        ui_messages::UiMessage::SetupPro(payload),
                        "failed to send pro setup message from server ws connection",
                    )
                    .await;
                }
                Message::Close(_) => {
                    info!("Client at {} disconnected.", who);
                    return;
                }
                _ => {
                    info!("Received non-text message: {:?}", msg);
                }
            }
        } else {
            info!("Client at {} disconnected.", who);

            return;
        }
    }
}
