use crate::AppHandle;
use anyhow::{Context, Result};
use tauri::{Window, WindowBuilder, WindowUrl, Wry};

pub fn setup(window: &Window<Wry>) {
    // open browser devtools automatically during development
    #[cfg(debug_assertions)]
    {
        window.open_devtools();
    }

    // Window vibrancy
    #[cfg(target_os = "macos")]
    {
        window_vibrancy::apply_vibrancy(
            window,
            window_vibrancy::NSVisualEffectMaterial::HudWindow,
            None,
            None,
        )
        .expect("Unsupported platform! 'apply_vibrancy' is only supported on macOS");
    }
    #[cfg(target_os = "windows")]
    {
        window_vibrancy::apply_blur(window, Some((18, 18, 18, 125)))
            .expect("Unsupported platform! 'apply_blur' is only supported on Windows");
    }
}

pub fn new_main(app_handle: &AppHandle, app_name: String) -> Result<()> {
    let handle = app_handle.clone();

    app_handle
        .run_on_main_thread(move || {
            // Config should match the config in `src-tauri/tauri.conf.json` for a consistent window
            // appearance
            let window_builder =
                WindowBuilder::new(&handle, "main".to_string(), WindowUrl::default())
                    .title(app_name)
                    .fullscreen(false)
                    .resizable(true)
                    .transparent(true)
                    .min_inner_size(1000.0, 700.0)
                    .inner_size(1200.0, 800.0)
                    .visible(false);

            #[cfg(target_os = "macos")]
            let window_builder = window_builder
                .title_bar_style(tauri::TitleBarStyle::Overlay)
                .hidden_title(true);

            let window = window_builder.build();

            if let Ok(window) = window {
                setup(&window);
            }
        })
        .context("Failed to create main window")
}
