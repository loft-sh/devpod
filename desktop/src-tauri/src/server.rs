use crate::{ui_messages, util, AppHandle, AppState, daemon};
use axum::{
    body::Body,
    extract::{
        connect_info::ConnectInfo,
        ws::{Message, WebSocket, WebSocketUpgrade},
        Path, Request, State as AxumState,
    },
    http::{HeaderMap, StatusCode},
    response::{IntoResponse, Response},
    routing::{any, get, post},
    Json, Router,
};
use http::Method;
use log::{debug, info, warn};
use serde::{Deserialize, Serialize};
use std::net::SocketAddr;
use tauri::Manager;
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
        .route("/daemon/:pro_id/status", get(daemon_status_handler))
        .route("/daemon/:pro_id/restart", get(daemon_restart_handler))
        .route("/daemon-proxy/:pro_id/*path", any(daemon_proxy_handler))
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
    AxumState(_server): AxumState<ServerState>,
    Json(payload): Json<SendSignalMessage>,
) -> impl IntoResponse {
    info!(
        "received request to send signal {} to process {}",
        payload.signal,
        payload.process_id.to_string()
    );
    util::kill_process(payload.process_id as u32);

    return StatusCode::OK;
}

async fn releases_handler(AxumState(server): AxumState<ServerState>) -> impl IntoResponse {
    let state = server.app_handle.state::<AppState>();
    let releases = state.releases.lock().unwrap();
    let releases = releases.clone();

    Json(releases)
}

async fn daemon_status_handler(
    Path(pro_id): Path<String>,
    AxumState(server): AxumState<ServerState>,
) -> impl IntoResponse {
    return match new_daemon_client(pro_id, server).await {
        Some(client) => match client.status().await {
            Ok(status) => Json(status).into_response(),
            Err(_) => StatusCode::INTERNAL_SERVER_ERROR.into_response(),
        },
        None => StatusCode::NOT_FOUND.into_response(),
    };
}

async fn new_daemon_client(
    pro_id: String,
    server: ServerState,
) -> Option<daemon::client::Client> {
    let state = server.app_handle.state::<AppState>();
    let pro = state.pro.read().await;
    return match pro.find_instance(pro_id) {
        Some(pro_instance) => match pro_instance.daemon() {
            Some(daemon) => Some(daemon.get_client().clone()),
            None => None,
        },
        None => None,
    };
}

async fn daemon_restart_handler(
    Path(pro_id): Path<String>,
    AxumState(server): AxumState<ServerState>,
) -> impl IntoResponse {
    let state = server.app_handle.state::<AppState>();
    let mut pro = state.pro.write().await;
    info!("Attempting to restart daemon");
    return match pro.find_instance_mut(pro_id) {
        Some(pro_instance) => match pro_instance.daemon_mut() {
            Some(daemon) => {
                daemon.try_stop().await;
                return StatusCode::OK.into_response();
            }
            None => StatusCode::NOT_FOUND.into_response(),
        },
        None => StatusCode::NOT_FOUND.into_response(),
    };
}

async fn daemon_proxy_handler(
    Path((pro_id, path)): Path<(String, String)>,
    AxumState(server): AxumState<ServerState>,
    mut req: Request<Body>,
) -> impl IntoResponse {
    return match new_daemon_client(pro_id, server).await {
        Some(client) => {
            // strip `daemon-proxy/:pro_id` from path before we hand the request to the daemon
            let original_query = req.uri().query();
            let new_path_with_query = match original_query {
                    Some(query) => format!("/{}?{}", path, query),
                    None => format!("/{}", path),
                };
                let mut parts = req.uri().clone().into_parts();
                parts.path_and_query = Some(new_path_with_query.parse().expect("Invalid path"));
                let new_uri = http::Uri::from_parts(parts).expect("Failed to build new URI");
                *req.uri_mut() = new_uri;

                let original_path = req.uri().path_and_query().expect("Invalid path").to_string();
                debug!("proxying daemon request: {}", original_path);

                return match client.proxy(req).await {
                    Ok(res) => res.into_response(),
                    Err(_) => StatusCode::INTERNAL_SERVER_ERROR.into_response(),
                };
            }
        None => StatusCode::NOT_FOUND.into_response(),
    };
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
