use serde::{Deserialize, Serialize};
use ts_rs::TS;
use lazy_static::lazy_static;
use tauri::image::Image;

pub mod client;

#[derive(Debug, Default, Serialize, Deserialize, Eq, PartialEq, TS)]
#[serde(rename_all = "camelCase", default)]
#[ts(export)]
pub struct DaemonStatus {
    pub state: DaemonState,
    pub online: bool,
    pub login_required: bool,
}

#[derive(Debug, Default, Eq, PartialEq, Serialize, Deserialize, TS)]
#[serde(rename_all = "lowercase")]
#[ts(export)]
pub enum DaemonState {
    Stopped,
    #[default]
    Pending,
    Running,
}
impl DaemonState {
    pub fn running_icon() -> Image<'static> {
        lazy_static! {
            static ref RUNNING_ICON: Image<'static> =
                Image::from_bytes(include_bytes!("../icons/running.png")).unwrap();
        }
        return RUNNING_ICON.clone();
    }
    pub fn stopped_icon() -> Image<'static> {
        lazy_static! {
            static ref STOPPED_ICON: Image<'static> =
                Image::from_bytes(include_bytes!("../icons/stopped.png")).unwrap();
        }
        return STOPPED_ICON.clone();
    }
    pub fn pending_icon() -> Image<'static> {
        lazy_static! {
            static ref PENDING_ICON: Image<'static> =
                Image::from_bytes(include_bytes!("../icons/pending.png")).unwrap();
        }
        return PENDING_ICON.clone();
    }

    pub fn get_icon(&self) -> tauri::image::Image {
        return match self {
            DaemonState::Running => Self::running_icon(),
            DaemonState::Stopped => Self::stopped_icon(),
            DaemonState::Pending => Self::pending_icon(),
        };
    }
}
