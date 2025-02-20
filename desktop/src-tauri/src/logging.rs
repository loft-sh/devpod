use log::LevelFilter;
use tauri::{plugin::TauriPlugin, Wry};
use tauri_plugin_log::{Target, TargetKind};

#[allow(unused_variables)]
pub fn build_plugin() -> TauriPlugin<Wry> {
    let enable_debug_logging: Option<&'static str> = option_env!("DEBUG");
    let mut targets: Vec<_> = vec![];

    #[cfg(debug_assertions)] // only enable during development
    if enable_debug_logging.is_some() {
        targets.push(Target::new(TargetKind::Stdout));
    }
    #[cfg(not(debug_assertions))] // only enable in release builds
    targets.push(Target::new(TargetKind::LogDir {
        file_name: Some("DevPod".to_string()),
    }));

    let builder = tauri_plugin_log::Builder::default().targets(targets);
    #[cfg(debug_assertions)] // only enable during development
    let builder = builder.level(LevelFilter::Debug);
    #[cfg(not(debug_assertions))] // only enable in release builds
    let builder = builder.level(LevelFilter::Info);

    builder.build()
}
