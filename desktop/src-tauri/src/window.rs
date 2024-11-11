use crate::AppHandle;
use anyhow::{Context, Result};
use log::error;
use tauri::{WebviewWindow, WebviewWindowBuilder, Wry, WebviewUrl};

#[derive(Clone, Debug)]
pub struct WindowHelper {
    app_handle: AppHandle,
}

impl WindowHelper {
    pub fn new(app_handle: AppHandle) -> Self {
        Self { app_handle }
    }

    pub fn setup(&self, window: &WebviewWindow<Wry>) {
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
                window_vibrancy::NSVisualEffectMaterial::Sidebar,
                None,
                None,
            )
            .expect("Unsupported platform! 'apply_vibrancy' is only supported on macOS");
        }
    }

    pub fn new_main(&self, app_name: String) -> Result<()> {
        let handle = self.app_handle.clone();
        let self_ = self.clone();

        #[cfg(target_os = "macos")]
        self.set_dock_icon_visibility(true);

        self.app_handle
            .run_on_main_thread(move || {
                // Config should match the config in `src-tauri/tauri.conf.json` for a consistent window appearance
                let window_builder =
                    WebviewWindowBuilder::new(&handle, "main".to_string(), WebviewUrl::default())
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
                    self_.setup(&window);
                }
            })
            .context("Failed to create main window")
    }

    pub fn new_update_ready_window(&self) -> Result<()> {
        let handle = self.app_handle.clone();

        self.app_handle
            .run_on_main_thread(move || {
                let window_builder = WebviewWindowBuilder::new(
                    &handle,
                    "update_ready".to_string(),
                    WebviewUrl::App("update-window/index.html".into()),
                )
                .title("DevPod Update")
                .fullscreen(false)
                .resizable(false)
                .transparent(false)
                .inner_size(300.0, 175.0)
                .visible(true);

                if let Err(err) = window_builder.build() {
                    error!("Failed to create update ready window: {}", err);
                }
            })
            .context("Failed to create update ready window")
    }
}

#[cfg(target_os = "macos")]
use cocoa::{
    appkit::{
        NSApp, NSApplicationActivateIgnoringOtherApps,
        NSApplicationActivationPolicy::{self, *},
        NSImage,
    },
    base::{id, nil, BOOL},
    foundation::{NSArray, NSData, NSString},
};
#[cfg(target_os = "macos")]
use dispatch::Queue;
#[cfg(target_os = "macos")]
use std::time::Duration;
#[cfg(target_os = "macos")]
use tauri::ActivationPolicy;

#[cfg(target_os = "macos")]
#[allow(dead_code)]
impl WindowHelper {
    unsafe fn get_current_application() -> cocoa::base::id {
        msg_send![class!(NSRunningApplication), currentApplication]
    }

    unsafe fn is_current_app_active() -> bool {
        let current_app = Self::get_current_application();
        #[cfg(not(target_arch = "aarch64"))]
        {
            let is_active: BOOL = msg_send![current_app, isActive];

            return is_active == cocoa::base::YES;
        }

        #[cfg(target_arch = "aarch64")]
        {
            let is_active: BOOL = msg_send![current_app, isActive];

            return is_active;
        }
    }

    pub fn set_dock_icon_visibility(&self, visible: bool) {
        unsafe {
            let psn = ProcessSerialNumber {
                lowLongOfPSN: K_CURRENT_PROCESS,
                highLongOfPSN: 0,
            };

            if !visible {
                let result = TransformProcessType(
                    &psn as *const ProcessSerialNumber,
                    TransformState::ProcessTransformToUIElementApplication,
                );
                if result != 0 {
                    error!("Failed to set dock icon visibility: {}", result);
                }
                return;
            }

            let is_active = Self::is_current_app_active();
            if is_active {
                let application_id = NSString::alloc(nil).init_str("com.apple.dock");
                let running_applications: id = msg_send![
                    class!(NSRunningApplication),
                    runningApplicationsWithBundleIdentifier: application_id
                ];
                for i in 0..NSArray::count(running_applications) {
                    let app: id = msg_send![running_applications, objectAtIndex: i];
                    let _: () = msg_send![
                        app,
                        activateWithOptions: NSApplicationActivateIgnoringOtherApps
                    ];
                    break;
                }

                Queue::main().exec_after(Duration::from_millis(1), move || {
                    let result = TransformProcessType(
                        &psn as *const ProcessSerialNumber,
                        TransformState::ProcessTransformToForegroundApplication,
                    );
                    if result != 0 {
                        error!("Failed to set dock icon visibility: {}", result);
                    }

                    Queue::main().exec_after(Duration::from_millis(1), move || {
                        let _: () = msg_send![
                            Self::get_current_application(),
                            activateWithOptions: NSApplicationActivateIgnoringOtherApps
                        ];
                    });
                });
            } else {
                let result = TransformProcessType(
                    &psn as *const ProcessSerialNumber,
                    TransformState::ProcessTransformToForegroundApplication,
                );
            }
            self.set_default_app_icon();
        }
    }

    fn set_default_app_icon(&self) {
        let _ = self.app_handle.run_on_main_thread(move || unsafe {
            let app: id = msg_send![class!(NSApplication), sharedApplication];
            let icon = include_bytes!("../icons/icon.icns");
            let data = NSData::dataWithBytes_length_(
                nil,
                icon.as_ptr() as *const std::os::raw::c_void,
                icon.len() as u64,
            );
            let app_icon = NSImage::initWithData_(NSImage::alloc(nil), data);

            let res: () = msg_send![app, setApplicationIconImage: app_icon];
        });
    }

    // May work at some point, will keep function here for now
    fn set_activation_policy(act_pol: ActivationPolicy) {
        let act_pol = match act_pol {
            ActivationPolicy::Regular => NSApplicationActivationPolicyRegular,
            ActivationPolicy::Accessory => NSApplicationActivationPolicyAccessory,
            ActivationPolicy::Prohibited => NSApplicationActivationPolicyProhibited,
            _ => unimplemented!(),
        };

        unsafe {
            let ns_app = NSApp();
            let res: BOOL = msg_send![ns_app, setActivationPolicy: act_pol];
        }
    }

    // May work at some point, will keep function here for now
    pub fn get_activation_policy() -> ActivationPolicy {
        unsafe {
            let ns_app = NSApp();
            let res: NSApplicationActivationPolicy = msg_send![ns_app, activationPolicy];
            match res {
                NSApplicationActivationPolicyRegular => ActivationPolicy::Regular,
                NSApplicationActivationPolicyAccessory => ActivationPolicy::Accessory,
                NSApplicationActivationPolicyProhibited => ActivationPolicy::Prohibited,
                _ => unimplemented!(),
            }
        }
    }
}
#[cfg(target_os = "macos")]
#[repr(u32)]
enum TransformState {
    // https://developer.apple.com/documentation/applicationservices/1501117-anonymous/kprocesstransformtoforegroundapplication?language=objc
    ProcessTransformToForegroundApplication = 1,
    // https://developer.apple.com/documentation/applicationservices/1501117-anonymous/kprocesstransformtouielementapplication?language=objc
    ProcessTransformToUIElementApplication = 4,
}

#[cfg(target_os = "macos")]
const K_CURRENT_PROCESS: u32 = 2;

#[cfg(target_os = "macos")]
#[allow(non_snake_case)]
#[repr(C)]
#[derive(Clone, Copy, Debug, Default, Eq, Hash, PartialEq)]
pub struct ProcessSerialNumber {
    pub highLongOfPSN: u32,
    pub lowLongOfPSN: u32,
}

#[cfg(target_os = "macos")]
#[link(name = "ApplicationServices", kind = "framework")]
extern "C" {
    fn TransformProcessType(psn: *const ProcessSerialNumber, transformState: TransformState)
        -> i32;
}
