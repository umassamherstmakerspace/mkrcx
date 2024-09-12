use std::collections::HashMap;

use reqwest::Client;
use serde::{Deserialize, Serialize};
use url::Url;

#[derive(Debug, Clone)]
pub struct HttpResponse<T> {
    pub status: u16,
    pub body: T,
}

pub type SerdeHttpResult<T> = Result<HttpResponse<T>, reqwest::Error>;
pub type HttpResult = SerdeHttpResult<String>;

#[derive(Debug, Clone, Serialize)]
pub struct LeashListOptions {
    pub offset: u32,
    pub limit: u32,
    pub include_deleted: bool,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct LeashListResult<T> {
    pub total: u32,
    pub data: Vec<T>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum LeashAuthenticator {
    Token(String),
    ApiKey(String),
}

impl LeashAuthenticator {
    pub fn get_authentication_header(&self) -> String {
        match self {
            LeashAuthenticator::Token(token) => format!("Bearer {}", token),
            LeashAuthenticator::ApiKey(api_key) => format!("API-Key {}", api_key),
        }
    }
}

#[derive(Debug, Clone)]
pub struct LeashClient {
    pub authenticator: LeashAuthenticator,
    pub base_url: String,
}

impl LeashClient {
    pub fn new(authenticator: LeashAuthenticator, base_url: String) -> Self {
        Self {
            authenticator,
            base_url,
        }
    }

    pub fn get_base_url(&self) -> &str {
        &self.base_url
    }

    pub async fn get(&self, path: &str, args: Option<HashMap<String, String>>) -> HttpResult {
        let client = Client::new();
        let url = Url::parse_with_params(
            &format!("{}/{}", self.base_url, path),
            args.unwrap_or_default(),
        )
        .unwrap();
        let response = client
            .get(&url.to_string())
            .header(
                "Authorization",
                self.authenticator.get_authentication_header(),
            )
            .send()
            .await?;
        let status = response.status().as_u16();
        let body = response.text().await?;
        Ok(HttpResponse { status, body })
    }

    pub async fn post(
        &self,
        path: &str,
        args: Option<HashMap<String, String>>,
        body: &str,
    ) -> HttpResult {
        let client = Client::new();
        let url = Url::parse_with_params(
            &format!("{}/{}", self.base_url, path),
            args.unwrap_or_default(),
        )
        .unwrap();
        let response = client
            .post(&url.to_string())
            .header(
                "Authorization",
                self.authenticator.get_authentication_header(),
            ).header("content-type", "application/json")
            .body(body.to_string())
            .send()
            .await?;
        let status = response.status().as_u16();
        let body = response.text().await?;
        Ok(HttpResponse { status, body })
    }

    pub async fn put(
        &self,
        path: &str,
        args: Option<HashMap<String, String>>,
        body: &str,
    ) -> HttpResult {
        let client = Client::new();
        let url = Url::parse_with_params(
            &format!("{}/{}", self.base_url, path),
            args.unwrap_or_default(),
        )
        .unwrap();

        let response = client
            .put(&url.to_string())
            .header(
                "Authorization",
                self.authenticator.get_authentication_header(),
            ).header("content-type", "application/json")
            .body(body.to_string())
            .send()
            .await?;
        let status = response.status().as_u16();
        let body = response.text().await?;
        Ok(HttpResponse { status, body })
    }

    pub async fn delete(&self, path: &str, args: Option<HashMap<String, String>>) -> HttpResult {
        let client = Client::new();
        let url = Url::parse_with_params(
            &format!("{}/{}", self.base_url, path),
            args.unwrap_or_default(),
        )
        .unwrap();
        let response = client
            .delete(&url.to_string())
            .header(
                "Authorization",
                self.authenticator.get_authentication_header(),
            )
            .send()
            .await?;
        let status = response.status().as_u16();
        let body = response.text().await?;
        Ok(HttpResponse { status, body })
    }

    pub async fn patch(
        &self,
        path: &str,
        args: Option<HashMap<String, String>>,
        body: &str,
    ) -> HttpResult {
        let client = Client::new();
        let url = Url::parse_with_params(
            &format!("{}/{}", self.base_url, path),
            args.unwrap_or_default(),
        )
        .unwrap();
        let response = client
            .patch(&url.to_string())
            .header(
                "Authorization",
                self.authenticator.get_authentication_header(),
            ).header("content-type", "application/json")
            .body(body.to_string())
            .send()
            .await?;
        let status = response.status().as_u16();
        let body = response.text().await?;
        Ok(HttpResponse { status, body })
    }

    pub async fn list<T>(
        &self,
        path: &str,
        limit: u32,
        offset: u32,
        args: Option<HashMap<String, String>>,
    ) -> SerdeHttpResult<LeashListResult<T>>
    where
        T: serde::de::DeserializeOwned,
    {
        let mut args = args.unwrap_or_default();
        args.insert("limit".to_string(), limit.to_string());
        args.insert("offset".to_string(), offset.to_string());
        let HttpResponse { status, body } = self.get(path, Some(args)).await?;
        let result: LeashListResult<T> = serde_json::from_str(&body).unwrap();
        Ok(HttpResponse {
            status,
            body: result,
        })
    }

    pub async fn list_all<T>(&self, path: &str) -> Result<Vec<T>, reqwest::Error>
    where
        T: serde::de::DeserializeOwned,
    {
        let mut offset = 0;
        let limit = 100;
        let mut result: Vec<T> = vec![];
        let mut current_result: LeashListResult<T>;
        loop {
            current_result = self.list(path, limit, offset, None).await?.body;
            result.extend(current_result.data);
            offset += limit;
            if current_result.total <= offset {
                break;
            }
        }
        Ok(result)
    }
}
