use log::LevelFilter;
use tauri::{plugin::TauriPlugin, Wry};
use tauri_plugin_log::LogTarget;

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

    let builder = tauri_plugin_log::Builder::default().targets(targets);

    #[cfg(debug_assertions)] // only enable during development
    let builder = builder.level(LevelFilter::Debug);

    #[cfg(not(debug_assertions))] // only enable in release builds
    let builder = builder.level(LevelFilter::Info);

    builder.build()
}
