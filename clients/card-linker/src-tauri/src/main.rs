// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::sync::Arc;

use keyring::Entry;
use leash_client::{
    client::{LeashAuthenticator, LeashClient},
    user::User,
};
use serde::{Deserialize, Serialize};
use tauri::{async_runtime::Mutex, Error, Result};

use card_helper::reader::AsyncPcsc;

const KEYSTORE_SERVICE: &str = "leash-card-linker";

type Leash = Arc<Mutex<LeashClient>>;
type ActiveUser = Arc<Mutex<Option<User>>>;
type Reader = Arc<Mutex<AsyncPcsc>>;

fn map_keyring_error(e: keyring::Error) -> Error {
    Error::Io(std::io::Error::new(std::io::ErrorKind::Other, e))
}

fn get_keystores() -> Result<(Entry, Entry)> {
    let apikey_entry = Entry::new(KEYSTORE_SERVICE, "apikey").map_err(map_keyring_error)?;
    let base_url_entry = Entry::new(KEYSTORE_SERVICE, "base_url").map_err(map_keyring_error)?;

    Ok((apikey_entry, base_url_entry))
}

fn get_leash_api() -> Result<LeashClient> {
    let apikey;
    let base_url;

    match env!("LEASH_SKIP_STORE") {
        "true" => {
            apikey = env!("LEASH_APIKEY").to_owned();
            base_url = env!("LEASH_URL").to_owned();
        },
        "false" => {
            let (apikey_entry, base_url_entry) = get_keystores()?;
            apikey = apikey_entry.get_password().unwrap_or_default();
            base_url = base_url_entry.get_password().unwrap_or_default();
        },
        _ => unreachable!()
    }

    Ok(LeashClient::new(
        LeashAuthenticator::ApiKey(apikey),
        base_url,
    ))
}

#[tauri::command]
async fn set_reader(
    reader_state: tauri::State<'_, Reader>,
) -> Result<()> {
    let mut ctx = reader_state.lock().await;
    let readers = ctx.list_readers().await.unwrap();
    ctx.set_reader(&readers[0]);

    Ok(())
}

#[tauri::command]
async fn get_card_id(
    reader_state: tauri::State<'_, Reader>,
) -> Result<String> {
    let ctx = reader_state.lock().await;
    let card = ctx.connect().await.unwrap();
    Ok(ctx.get_card_id(&card).await.unwrap())
}

#[tauri::command]
async fn get_checkin(
    token: String,
    api_state: tauri::State<'_, Leash>,
    user_state: tauri::State<'_, ActiveUser>,
) -> Result<()> {
    let checkin = {
        let api = api_state.lock().await;
        api.get(&format!("api/users/get/checkin/{}", token), None)
            .await
    };

    match checkin {
        Ok(checkin) => {
            let body = checkin.body;
            let p: User = serde_json::from_str(&body)?;

            {
                let mut user = user_state.lock().await;
                *user = Some(p);
            }

            Ok(())
        }
        Err(e) => Err(Error::Io(std::io::Error::new(std::io::ErrorKind::Other, e))),
    }
}
#[tauri::command]
async fn get_user(
    user_state: tauri::State<'_, ActiveUser>,
) -> Result<(String, String)> {
    let user = match user_state.lock().await.clone() {
        Some(user) => user,
        None => {
            return Err(tauri::Error::FailedToExecuteApi(
                tauri::api::Error::Command("get_user".to_string()),
            ))
        }
    };

    Ok((user.name, user.email))
}

#[tauri::command]
async fn set_card(
    card_number: String,
    api_state: tauri::State<'_, Leash>,
    user_state: tauri::State<'_, ActiveUser>,
) -> Result<()> {
    let user = match user_state.lock().await.clone() {
        Some(user) => user,
        None => {
            return Err(tauri::Error::FailedToExecuteApi(
                tauri::api::Error::Command("set_card".to_string()),
            ))
        }
    };

    let checkin = {
        let card = format!("{{\"card_id\": \"{}\"}}", card_number);
        let api = api_state.lock().await;
        api.patch(&format!("api/users/{}", user.id), None, &card)
            .await
    };

    match checkin {
        Ok(_) => {
            let mut u = user_state.lock().await;
            *u = None;
            Ok(())
        }
        Err(e) => Err(Error::Io(std::io::Error::new(std::io::ErrorKind::Other, e))),
    }
}

#[tauri::command]
async fn clear_user(user_state: tauri::State<'_, ActiveUser>) -> Result<()> {
    let mut user = user_state.lock().await;
    *user = None;
    Ok(())
}

#[tauri::command]
async fn login(base_url: String, apikey: String, state: tauri::State<'_, Leash>) -> Result<()> {
    let (apikey_entry, base_url_entry) = get_keystores()?;
    apikey_entry
        .set_password(&apikey)
        .map_err(map_keyring_error)?;
    base_url_entry
        .set_password(&base_url)
        .map_err(map_keyring_error)?;

    let mut api = state.lock().await;
    api.base_url = base_url;
    api.authenticator = LeashAuthenticator::ApiKey(apikey);

    Ok(())
}

fn main() {
    tauri::Builder::default()
        .manage(Arc::new(Mutex::new(get_leash_api().unwrap())))
        .manage(Arc::new(Mutex::new(None::<User>)))
        .manage(Arc::new(Mutex::new(AsyncPcsc::establish().unwrap())))
        .invoke_handler(tauri::generate_handler![
            get_checkin,
            get_user,
            set_card,
            set_reader,
            get_card_id,
            clear_user,
            login
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
