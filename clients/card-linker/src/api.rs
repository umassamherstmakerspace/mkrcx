use keyring::Entry;
use leash_client::client::{LeashAuthenticator, LeashClient};

const KEYSTORE_SERVICE: &str = "leash-card-linker";

fn get_keystores() -> keyring::Result<(Entry, Entry)> {
    let apikey_entry = Entry::new(KEYSTORE_SERVICE, "apikey")?;
    let base_url_entry = Entry::new(KEYSTORE_SERVICE, "base_url")?;

    Ok((apikey_entry, base_url_entry))
}

pub fn get_leash_api() -> keyring::Result<LeashClient> {
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