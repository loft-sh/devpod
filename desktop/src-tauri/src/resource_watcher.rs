use crate::{
    commands::{
        list_pro_instances::ListProInstancesCommand, list_workspaces::ListWorkspacesCommand,
        start_daemon::StartDaemonCommand, DevpodCommandError,
    },
    daemon,
    system_tray::{ToSystemTraySubmenu, SYSTEM_TRAY_ICON_BYTES, WARNING_SYSTEM_TRAY_ICON_BYTES},
    ui_messages,
};
use crate::{AppHandle, AppState};
use anyhow::anyhow;
use dirs::home_dir;
use log::{debug, error, info};
use serde::Deserialize;
use std::{collections::HashSet, hash::Hash, time};
use tauri::{
    async_runtime::Receiver,
    image::Image,
    menu::{IconMenuItem, MenuItem, Submenu, SubmenuBuilder},
    Manager, Wry,
};
use tauri_plugin_notification::NotificationExt;
use tauri_plugin_shell::process::{CommandChild, CommandEvent};


pub trait Identifiable {
    type ID: Eq + Hash + Clone;
    fn id(&self) -> Self::ID;
}
#[derive(Default)]
pub struct WorkspacesState {
    workspaces: Vec<Workspace>,
    submenu: Option<Submenu<Wry>>,
}

#[derive(Deserialize, Clone)]
pub struct Workspace {
    id: String,
    #[serde(skip)]
    menu_item: Option<MenuItem<Wry>>,
}
impl Identifiable for Workspace {
    type ID = String;
    fn id(&self) -> String {
        return self.id.clone();
    }
}
impl PartialEq for Workspace {
    fn eq(&self, other: &Self) -> bool {
        self.id() == other.id()
    }
}
impl Workspace {
    fn new_menu_item(&self, app_handle: &AppHandle) -> tauri::Result<MenuItem<Wry>> {
        return MenuItem::with_id(
            app_handle,
            WorkspacesState::item_id(&self.id()),
            self.id(),
            true,
            None::<&str>,
        );
    }
}

impl WorkspacesState {
    pub const IDENTIFIER_PREFIX: &'static str = "workspaces-";
    pub const CREATE_WORKSPACE_ID: &'static str = "workspaces-create_workspace";

    fn item_id(id: &String) -> String {
        format!("{}{}", Self::IDENTIFIER_PREFIX, id)
    }

    pub fn set_submenu(&mut self, submenu: Submenu<Wry>) {
        self.submenu = Some(submenu);
    }

    pub async fn load_workspaces(
        app_handle: &AppHandle,
    ) -> Result<Vec<Workspace>, DevpodCommandError> {
        let list_workspaces_cmd = ListWorkspacesCommand::new();

        return list_workspaces_cmd.exec(app_handle).await;
    }
}

impl ToSystemTraySubmenu for WorkspacesState {
    fn to_submenu(&self, app_handle: &AppHandle) -> anyhow::Result<tauri::menu::Submenu<Wry>> {
        let mut submenu = SubmenuBuilder::with_id(app_handle, "workspace", "Workspaces");

        let create_workspace = MenuItem::with_id(
            app_handle,
            Self::CREATE_WORKSPACE_ID,
            "Create Workspace",
            true,
            None::<&str>,
        )?;
        submenu = submenu.item(&create_workspace);
        submenu = submenu.separator();

        return Ok(submenu.build()?);
    }
}

static CAPABILITY_DAEMON: &str = "daemon";
static MAX_RETRY_COUNT: i64 = 10;
static RETRY_DEBUG_THRESHOLD: i64 = 7;
#[derive(Default)]
pub struct ProState {
    instances: Vec<ProInstance>,
    submenu: Option<Submenu<tauri::Wry>>,
    all_ready: bool,
}
impl Identifiable for ProInstance {
    type ID = String;
    fn id(&self) -> String {
        return self.host.clone();
    }
}
impl PartialEq for ProInstance {
    fn eq(&self, other: &Self) -> bool {
        self.id() == other.id()
    }
}

impl ProState {
    pub const IDENTIFIER_PREFIX: &'static str = "pro-instances-";

    fn item_id(id: &String) -> String {
        format!("{}{}", Self::IDENTIFIER_PREFIX, id)
    }

    pub async fn load_pro_instances(
        app_handle: &AppHandle,
    ) -> Result<Vec<ProInstance>, DevpodCommandError> {
        let cmd = ListProInstancesCommand::new();
        let pro_instances = cmd.exec(app_handle).await?;

        Ok(pro_instances)
    }

    pub fn set_submenu(&mut self, submenu: Submenu<Wry>) {
        self.submenu = Some(submenu);
    }

    pub fn find_instance(&self, pro_id: String) -> Option<&ProInstance> {
        return self.instances.iter().find(|i| i.id() == pro_id);
    }

    pub fn find_instance_mut(&mut self, pro_id: String) -> Option<&mut ProInstance> {
        return self.instances.iter_mut().find(|i| i.id() == pro_id);
    }
}

#[derive(Deserialize)]
#[serde(rename_all(serialize = "camelCase", deserialize = "camelCase"))]
pub struct ProInstance {
    host: String,
    provider: Option<String>,
    context: Option<String>,
    capabilities: Option<Vec<String>>,
    #[serde(skip)]
    menu_item: Option<IconMenuItem<Wry>>,
    #[serde(skip)]
    daemon: Option<Daemon>,
}
impl ProInstance {
    pub fn has_capability(&self, capability: String) -> bool {
        if let Some(capabilities) = &self.capabilities {
            for cap in capabilities {
                if *cap == capability {
                    return true;
                }
            }
        }
        return false;
    }

    pub fn daemon(&self) -> &Option<Daemon> {
        return &self.daemon;
    }

    pub fn daemon_mut(&mut self) -> Option<&mut Daemon> {
        return self.daemon.as_mut();
    }

    fn new_menu_item(&self, app_handle: &AppHandle) -> tauri::Result<IconMenuItem<Wry>> {
        return IconMenuItem::with_id(
            app_handle,
            ProState::item_id(&self.id()),
            self.id(),
            true,
            self.get_icon(),
            None::<&str>,
        );
    }

    fn get_icon(&self) -> Option<Image> {
        return self
            .daemon
            .as_ref()
            .map(|daemon| daemon.status.state.get_icon());
    }
}

#[derive(Debug)]
pub struct Daemon {
    status: daemon::DaemonStatus,
    command: Option<(Receiver<CommandEvent>, CommandChild)>,
    retry_count: i64,
    client: daemon::client::Client,
    provider: Option<String>,

    notified_user_daemon_failed: bool,
    notified_login_required: bool,
}
impl Daemon {
    pub fn new(context: Option<String>, provider: Option<String>) -> anyhow::Result<Daemon> {
        let socket_addr = Daemon::get_socket_addr(context.clone(), provider.clone())?;
        let client = daemon::client::Client::new(socket_addr);

        return Ok(Daemon {
            status: daemon::DaemonStatus::default(),
            command: None,
            retry_count: 0,
            notified_user_daemon_failed: false,
            notified_login_required: false,
            provider,
            client,
        });
    }

    fn get_socket_addr(
        context: Option<String>,
        provider: Option<String>,
    ) -> Result<String, DevpodCommandError> {
        let provider = provider.clone().ok_or(DevpodCommandError::Any(anyhow!(
            "provider not set for pro instance"
        )))?;
        #[cfg(unix)]
        {
            let home = Self::get_home()?;
            let context = context.clone().unwrap_or("default".to_string());

            return Ok(format!(
                "{}/contexts/{}/providers/{}/daemon/devpod.sock",
                home, context, provider
            ));
        }
        #[cfg(windows)]
        {
            return Ok(format!("\\\\.\\pipe\\devpod.{}", provider).to_string());
        }
    }

    fn get_home() -> anyhow::Result<String> {
        if let Ok(devpod_home) = std::env::var("DEVPOD_HOME") {
            return Ok(devpod_home);
        }

        if let Some(mut home) = home_dir() {
            home.push(".devpod");
            if let Some(home) = home.to_str() {
                return Ok(home.to_owned());
            }
        }

        return Err(anyhow!("Failed to get home directory for current user"));
    }

    pub fn status(&self) -> &daemon::DaemonStatus {
        return &self.status;
    }

    pub async fn get_status(&self) -> anyhow::Result<daemon::DaemonStatus> {
        return self.client.status().await;
    }

    pub async fn proxy_request(
        &self,
        req: daemon::client::Request,
    ) -> anyhow::Result<daemon::client::Response> {
        let path = req.uri().path_and_query();
        debug!("proxying daemon request: {}", path.expect("Invalid path"));

        return self.client.proxy(req).await;
    }

    pub async fn try_start(&mut self, host: String, app_handle: &AppHandle) {
        info!("[{}] attempting to start daemon", host.clone());
        if let Some(_) = self.command {
            self.try_stop().await;
        }
        match self.spawn(host.clone(), app_handle).await {
            Ok(command) => {
                info!("[{}] Successfully started daemon", host.clone());
                self.command = Some(command);
            }
            Err(err) => {
                error!("[{}] Failed to spawn daemon command {:?}", host, err);
            }
        }
    }

    pub async fn try_stop(&mut self) {
        if let Some(command) = self.command.take() {
            let pid = command.1.pid();
            if let Err(err) = command.1.kill() {
                debug!("Failed to kill command {:?}", err);
                // kill it with fire
                crate::util::kill_process(pid);
            }
        }
        self.command = None;
        self.status = daemon::DaemonStatus::default();
    }

    fn should_retry(&mut self, app_handle: &AppHandle) -> bool {
        if self.status.login_required {
            return false;
        }

        self.retry_count += 1;
        if self.retry_count < MAX_RETRY_COUNT {
            return true;
        } else {
            self.try_notify_failed(app_handle);

            // fall back to every 5 ticks after reaching `MAX_RETRY_COUNT`
            return self.retry_count % 5 == 0;
        }
    }

    fn should_debug(&self) -> bool {
        return self.retry_count >= RETRY_DEBUG_THRESHOLD;
    }

    async fn spawn(
        &mut self,
        host: String,
        app_handle: &AppHandle,
    ) -> Result<(Receiver<CommandEvent>, CommandChild), DevpodCommandError> {
        let (mut rx, child) = StartDaemonCommand::new(host.clone(), self.should_debug())
            .command(app_handle)?
            .spawn()?;

        tokio::select! {
            status = self.get_initial_status(&mut rx) => {
                if let Ok(status) = status {
                    self.status = status;
                    if self.status.login_required {
                        self.try_notify_login(host, app_handle).await;
                    }
                }
            },
            _ = tokio::time::sleep(tokio::time::Duration::from_secs(30)) => {
                return Err(DevpodCommandError::Any(anyhow!("Timed out waiting for daemon to start")));
            }
        }

        return Ok((rx, child));
    }

    async fn get_initial_status(
        &self,
        rx: &mut Receiver<CommandEvent>,
    ) -> anyhow::Result<daemon::DaemonStatus> {
        loop {
            if let Some(event) = rx.recv().await {
                match event {
                    CommandEvent::Stdout(out) => {
                        return serde_json::from_slice::<daemon::DaemonStatus>(&out)
                            .map_err(|err| anyhow!("failed to parse status: {:?}", err));
                    }
                    _ => {
                        return Err(anyhow!("expected stdout message"));
                    }
                }
            }
        }
    }

    async fn try_notify_login(&mut self, host: String, app_handle: &AppHandle) {
        if self.notified_login_required {
            return;
        }

        let msg = ui_messages::LoginRequiredMsg {
            host,
            provider: self.provider.clone().unwrap_or("".to_string()),
        };
        let _ = app_handle
            .state::<AppState>()
            .ui_messages
            .send(ui_messages::UiMessage::LoginRequired(msg))
            .await;

        self.notified_login_required = true;
    }

    fn try_notify_failed(&mut self, app_handle: &AppHandle) {
        if self.notified_user_daemon_failed {
            return;
        }
        let res = app_handle
                .notification()
                .builder()
                .title("Failed to start daemon")
                .body("Please take a look at \"Settings > Open Logs\" or report this issue to an administrator")
                .show();
        if let Err(err) = res {
            error!("Unable to send daemon-failed notification: {}", err);
        }

        self.notified_user_daemon_failed = true;
    }
}

impl ToSystemTraySubmenu for ProState {
    fn to_submenu(&self, app_handle: &AppHandle) -> anyhow::Result<tauri::menu::Submenu<Wry>> {
        return Ok(SubmenuBuilder::with_id(app_handle, "pro", "Pro").build()?);
    }
}

pub fn setup(app_handle: &AppHandle) {
    let state = app_handle.state::<AppState>();
    let mut resource_handles = state.resources_handles.lock().unwrap();
    // daemon watcher
    let daemon_app_handle = app_handle.clone();
    let daemon_watcher_handle = tauri::async_runtime::spawn(async move {
        let sleep_duration = time::Duration::from_millis(1_000);
        loop {
            let res = watch_daemons(&daemon_app_handle).await;
            if let Err(err) = res {
                error!("watch daemons: {}", err)
            };
            let _ = tokio::time::sleep(sleep_duration).await;
        }
    });
    resource_handles.push(daemon_watcher_handle);

    // main resources watchers
    let resources_app_handle = app_handle.clone();
    let resources_handle = tauri::async_runtime::spawn(async move {
        let sleep_duration = time::Duration::from_millis(5_000);
        loop {
            handle_workspaces(&resources_app_handle).await;
            handle_pro_instances(&resources_app_handle).await;
            let _ = tokio::time::sleep(sleep_duration).await;
        }
    });
    resource_handles.push(resources_handle);
}

pub async fn shutdown(app_handle: &AppHandle) {
    info!("Shutting down resource watchers");
    let state = app_handle.state::<AppState>();
    // shut down background tasks
    let mut handles = state.resources_handles.lock().unwrap();
    for handle in handles.iter() {
        handle.abort();
    }
    handles.clear();
    // shut down daemons
    let mut pro_state = state.pro.write().await;
    for pro_instance in pro_state.instances.iter_mut() {
        let id = pro_instance.id().clone();
        if let Some(daemon) = pro_instance.daemon.as_mut() {
            info!("[{}] Stopping daemon", id);
            daemon.try_stop().await;
        }
    }
}

async fn watch_daemons(app_handle: &AppHandle) -> anyhow::Result<()> {
    let state = app_handle.state::<AppState>();
    let mut pro_state = state.pro.write().await;
    let mut all_ready = true;

    // TODO: parallelize
    for instance in &mut pro_state.instances {
        let id = instance.id();
        if !instance.has_capability(CAPABILITY_DAEMON.to_string()) {
            continue;
        }
        let daemon = instance.daemon();
        if daemon.is_none() {
            instance.daemon = Some(
                Daemon::new(instance.context.clone(), instance.provider.clone())
                    .map_err(|err| anyhow!("Failed to create new daemon: {}", err))?,
            );
        }
        let daemon = instance.daemon.as_mut().unwrap();
        if !daemon.should_retry(app_handle) {
            all_ready = false;
            continue;
        }

        if let Some(menu_item) = &instance.menu_item {
            let _ = menu_item.set_icon(Some(daemon.status.state.get_icon()));
        }
        match daemon.get_status().await {
            Ok(status) => {
                daemon.status = status;
                match daemon.status.state {
                    daemon::DaemonState::Running => {
                        daemon.retry_count = 0;
                        // reset login notification once the daemon is up and running
                        daemon.notified_login_required = false;
                    }
                    daemon::DaemonState::Stopped => {
                        all_ready = false;
                        info!("[{}] daemon stopped, attempting to restart", id);
                        daemon.status.state = daemon::DaemonState::Pending;
                        daemon.try_start(id, app_handle).await;
                    }
                    daemon::DaemonState::Pending => {
                        all_ready = false;
                    }
                }
            }
            Err(err) => {
                all_ready = false;
                info!("[{}] failed to get daemon status: {}", id, err);
                daemon.status.state = daemon::DaemonState::Stopped;

                match daemon.command.as_mut() {
                    Some(cmd) => {
                        // replay stderr for debugging purposes
                        while let Some(event) = cmd.0.recv().await {
                            if let CommandEvent::Stderr(out) = event {
                                error!("{}", String::from_utf8(out)?.trim());
                            }
                        }
                        // kill the current command and restart on the next iteration
                        crate::util::kill_process(cmd.1.pid());
                        daemon.command = None;
                    }
                    None => {
                        daemon.status.state = daemon::DaemonState::Pending;
                        daemon.try_start(id, app_handle).await;
                    }
                }
            }
        }

        if let Some(menu_item) = &instance.menu_item {
            let _ = menu_item.set_icon(Some(daemon.status.state.get_icon()));
        }
    }

    if pro_state.all_ready != all_ready {
        pro_state.all_ready = all_ready;

        // update main system tray icon
        if let Some(main_tray) = app_handle.tray_by_id("main") {
            let icon = match pro_state.all_ready {
                true => Image::from_bytes(SYSTEM_TRAY_ICON_BYTES).unwrap(),
                false => Image::from_bytes(WARNING_SYSTEM_TRAY_ICON_BYTES).unwrap(),
            };
            let _ = main_tray.set_icon(Some(icon));
            let _ = main_tray.set_icon_as_template(true);
        }
    }

    return Ok(());
}

async fn handle_workspaces(app_handle: &AppHandle) {
    let workspaces = WorkspacesState::load_workspaces(app_handle).await;
    if workspaces.is_err() {
        return;
    }

    let mut workspaces = workspaces.unwrap();
    let state = app_handle.state::<AppState>();
    let state = &mut state.workspaces.write().await;
    if workspaces == state.workspaces {
        return;
    }

    if let Some(submenu) = &state.submenu {
        let (removed, added) = diff_mut(&state.workspaces, &mut workspaces);
        for w in removed {
            if let Some(menu_item) = &w.menu_item {
                _ = submenu.remove(menu_item);
            }
        }
        for w in added {
            if let Ok(menu_item) = w.new_menu_item(app_handle) {
                let _ = submenu.append(&menu_item);
                w.menu_item = Some(menu_item);
            }
        }
    }
    state.workspaces = workspaces;
}

async fn handle_pro_instances(app_handle: &AppHandle) {
    let pro_instances = ProState::load_pro_instances(app_handle).await;
    if pro_instances.is_err() {
        return;
    }
    let mut pro_instances = pro_instances.unwrap();

    let state = app_handle.state::<AppState>();
    let state = &mut state.pro.write().await;
    if pro_instances == state.instances {
        return;
    }
    if let Some(submenu) = &state.submenu {
        let (removed, added) = diff_mut(&state.instances, &mut pro_instances);
        for p in removed {
            if let Some(menu_item) = &p.menu_item {
                _ = submenu.remove(menu_item);
            }
        }
        for p in added {
            if let Ok(menu_item) = p.new_menu_item(app_handle) {
                let _ = submenu.append(&menu_item);
                p.menu_item = Some(menu_item);
            }
        }
    }

    state.instances = pro_instances;
}

fn diff_mut<'a, T: Identifiable>(old: &'a [T], new: &'a mut [T]) -> (Vec<&'a T>, Vec<&'a mut T>) {
    let old_ids: HashSet<_> = old.iter().map(|item| item.id()).collect();
    let new_ids: HashSet<_> = new.iter().map(|item| item.id()).collect();

    let removed: Vec<_> = old
        .iter()
        .filter(|ws| !new_ids.contains(&ws.id()))
        .collect();

    let added: Vec<_> = new
        .iter_mut()
        .filter(|ws| !old_ids.contains(&ws.id()))
        .collect();

    return (removed, added);
}
