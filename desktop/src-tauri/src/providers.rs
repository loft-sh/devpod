use crate::commands::{delete_provider::DeleteProviderCommand, DevpodCommandConfig};
use crate::util::with_data_store;
use crate::AppHandle;
use log::info;

pub fn check_dangling_provider(app_handle: &AppHandle) {
    let dangling_provider_key = "danglingProviders"; // WARN: needs to match the key defined in typescript
    let filename = ".providers.json"; // WARN: needs to match the file name defined in typescript

    let _ = with_data_store(app_handle, filename, |store| {
        store
            .get(dangling_provider_key)
            .and_then(|dangling_providers| {
                serde_json::from_value::<Vec<String>>(dangling_providers.clone()).ok()
            })
            .map(|dangling_providers| {
                info!(
                    "Found dangling providers: {}, attempting to delete",
                    dangling_providers.join(", ")
                );

                for dangling_provider in dangling_providers.iter() {
                    if DeleteProviderCommand::new(dangling_provider.clone())
                        .exec()
                        .is_ok()
                        && store.delete(dangling_provider_key).is_ok()
                    {
                        info!(
                            "Successfully deleted dangling provider: {}",
                            dangling_provider
                        );
                        let _ = store.save();
                    }
                }
            });

        Ok(())
    });
}
