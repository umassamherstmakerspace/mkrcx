use leash_client::client::{LeashAuthenticator, LeashClient};

const KEYSTORE_SERVICE: &str = "leash-card-linker";


pub fn get_leash_api() -> LeashClient {
    let apikey;
    let base_url;

    apikey = env!("LEASH_APIKEY").to_owned();
    base_url = env!("LEASH_URL").to_owned();

    LeashClient::new(
        LeashAuthenticator::ApiKey(apikey),
        base_url,
    )
}