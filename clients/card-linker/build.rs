use std::{fs, path::Path};

use serde::{Deserialize, Serialize};

#[derive(Default, Serialize, Deserialize)]
struct LeashConfig {
    skip_store: bool,
    url: String,
    apikey: String,
}

#[derive(Default, Serialize, Deserialize)]
struct Config {
    leash: LeashConfig,
}

fn main() {
  let config_path = Path::new("config.toml");

    let config_text = if config_path.exists() {
        match fs::read_to_string(config_path) {
            Ok(c) => c,
            Err(_) => {
                panic!("Could not read file `{}`", config_path.to_string_lossy());
            }
        }
    } else {
        "".to_owned()
    };

    let config: Config = match toml::from_str(&config_text) {
        Ok(c) => c,
        Err(_) => {
            let config = Config::default();
            fs::write(config_path, toml::to_string(&config).unwrap()).unwrap();
            config
        }
    };

    println!("cargo:rustc-env=LEASH_URL={}", config.leash.url);
    println!("cargo:rustc-env=LEASH_APIKEY={}", config.leash.apikey);
    println!("cargo:rustc-env=LEASH_SKIP_STORE={}", config.leash.skip_store);
}
