use log::LevelFilter;
use tauri::{plugin::TauriPlugin, Wry};
use tauri_plugin_log::{LogTarget, LogLevel};

#[allow(unused_variables)]
pub fn build_plugin() -> TauriPlugin<Wry> {
    let enable_debug_logging: Option<&'static str> = option_env!("DEBUG");
    let mut targets: Vec<_> = vec![];

    #[cfg(debug_assertions)] // only enable during development
    if enable_debug_logging.is_some() {
        targets.push(LogTarget::Stdout);
    }

    #[cfg(not(debug_assertions))] // only enable in release builds
    targets.push(LogTarget::LogDir);

    tauri_plugin_log::Builder::default()
        .targets(targets)
        .level(LevelFilter::Debug)
        .build()
}
