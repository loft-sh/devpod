use crate::commands::{delete_provider::DeleteProviderCommand, DevpodCommandConfig};
use crate::AppHandle;
use log::info;
use tauri::Wry;

pub fn check_dangling_provider(app: &AppHandle) {
    use tauri::Manager;
    use tauri_plugin_store::{with_store, StoreCollection};

    let stores = app.state::<StoreCollection<Wry>>();
    let file_name = ".providers.dat"; // WARN: needs to match the file name defined in typescript
    let dangling_provider_key = "danglingProviders"; // WARN: needs to match the key defined in typescript
    let path = app.path_resolver().app_data_dir();
    if path.is_none() {
        return;
    }

    let mut path = path.expect("AppDataDir should exist");
    path.push(file_name);

    let _ = with_store(app.app_handle(), stores, path, |store| {
        store
            .get(dangling_provider_key)
            .and_then(|dangling_providers| {
                serde_json::from_value::<Vec<String>>(dangling_providers.clone()).ok()
            })
            .and_then(|dangling_providers| {
                info!(
                    "Found dangling providers: {}, attempting to delete",
                    dangling_providers.join(", ")
                );

                for dangling_provider in dangling_providers.iter() {
                    if DeleteProviderCommand::new(dangling_provider.clone())
                        .exec()
                        .is_ok()
                    {
                        if let Ok(_) = store.delete(dangling_provider_key) {
                            info!(
                                "Successfully deleted dangling provider: {}",
                                dangling_provider
                            );
                            let _ = store.save();
                        };
                    }
                }

                Some(())
            });

        Ok(())
    });
}
