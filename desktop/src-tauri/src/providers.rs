use crate::commands::delete_pro_instance::DeleteProInstanceCommand;
use crate::commands::list_pro_instances::ListProInstancesCommand;
use crate::commands::{delete_provider::DeleteProviderCommand, DevpodCommandConfig};
use crate::resource_watcher::{Identifiable, ProInstance};
use crate::AppHandle;
use log::{debug, error, info};
use tauri_plugin_store::StoreExt;

pub fn check_dangling_provider(app_handle: &AppHandle) {
    let dangling_provider_key = "danglingProviders"; // WARN: needs to match the key defined in typescript
    let filename = ".providers.json"; // WARN: needs to match the file name defined in typescript

    debug!("Checking for dangling providers");
    let store = app_handle.store(filename);
    if store.is_err() {
        error!("unable to open store {}", filename);
        return;
    }
    let store = store.unwrap();
    let dangling_providers = store
        .get(dangling_provider_key)
        .and_then(|dangling_providers| {
            serde_json::from_value::<Vec<String>>(dangling_providers.clone()).ok()
        });

    if dangling_providers.is_none() {
        debug!("No dangling providers found");
        return;
    }
    let dangling_providers = dangling_providers.unwrap();

    if dangling_providers.is_empty() {
        debug!("No dangling providers found");
        return;
    }

    info!(
        "Found dangling providers: {}, attempting to delete",
        dangling_providers.join(", ")
    );

    let pro_instances = match ListProInstancesCommand::new().exec_blocking(app_handle) {
        Ok(pro_instances) => pro_instances,
        Err(err) => {
            error!("Failed to list pro instances, {}", err);
            vec![]
        }
    };

    for dangling_provider in dangling_providers.iter() {
        // Make sure we clean up accompanying pro instances
        check_pro_instances(app_handle, &pro_instances, &dangling_provider);

        if DeleteProviderCommand::new(dangling_provider.clone())
            .exec_blocking(&app_handle)
            .is_ok()
            && store.delete(dangling_provider_key)
        {
            info!(
                "Successfully deleted dangling provider: {}",
                dangling_provider
            );
            let _ = store.save();
        }
    }
}

fn check_pro_instances(
    app_handle: &AppHandle,
    pro_instances: &Vec<ProInstance>,
    dangling_provider: &String,
) {
    if let Some(pro_instance) = pro_instances
        .iter()
        .find(|pro_instance| &pro_instance.id() == dangling_provider)
    {
        let pro_id = pro_instance.id();
        info!(
            "Found dangling provider {} matching pro instance {}",
            dangling_provider, pro_id
        );

        match DeleteProInstanceCommand::new(pro_id.to_string()).exec_blocking(app_handle) {
            Ok(_) => info!("Successfully deleted pro instance {}", pro_id),
            Err(err) => error!("Failed to delete pro instance {}, {}", pro_id, err),
        }
    }
}
