use crate::{ui_messages, AppHandle, AppState};
use axum::{
    extract::{
        connect_info::ConnectInfo,
        ws::{Message, WebSocket, WebSocketUpgrade},
    },
    http::{HeaderMap, StatusCode},
    response::{IntoResponse, Response},
    routing::get,
    Json, Router,
};
use http::Method;
use log::{info, warn};
use std::net::SocketAddr;
use tauri::{Manager, State};
use tower_http::cors::{Any, CorsLayer};

pub async fn setup(app_handle: &AppHandle) -> anyhow::Result<()> {
    let handle = app_handle.clone();
    let handle_releases = app_handle.clone();
    let cors = CorsLayer::new()
        .allow_methods([Method::GET, Method::POST])
        .allow_origin(Any);

    let router = Router::new()
        .route(
            "/ws",
            get(move |upgrade, headers, info| ws_handler(upgrade, headers, info, handle.clone())),
        )
        .route("/releases", get(move || releases_handler(handle_releases)))
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
async fn releases_handler(app_handle: AppHandle) -> impl IntoResponse {
    #[cfg(feature = "enable-updater")]
    {
        let state = app_handle.state::<AppState>();
        let releases = state.releases.lock().unwrap();
        let releases = releases.clone();

        Json(releases)
    }

    #[cfg(not(feature = "enable-updater"))]
    {
        (StatusCode::NOT_FOUND, "Not found")
    }
}

async fn ws_handler(
    ws: WebSocketUpgrade,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    app_handle: AppHandle,
) -> Response {
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
