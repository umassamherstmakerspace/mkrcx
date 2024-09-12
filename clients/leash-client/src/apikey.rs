// export class APIKey {
// 	private api: LeashAPI;
// 	key: string;
// 	createdAt: Date;
// 	updatedAt: Date;
// 	deletedAt?: Date;

// 	private userID: number;
// 	description: string;
// 	fullAccess: boolean;
// 	permissions: string[];

// 	private endpointPrefix: string;

// 	constructor(api: LeashAPI, key: LeashAPIKey, endpointPrefix: string) {
// 		this.api = api;
// 		this.key = key.Key;
// 		this.createdAt = new Date(key.CreatedAt);
// 		this.updatedAt = new Date(key.UpdatedAt);
// 		if (key.DeletedAt) {
// 			this.deletedAt = new Date(key.DeletedAt);
// 		}

// 		this.userID = key.UserID;
// 		this.description = key.Description;
// 		this.fullAccess = key.FullAccess;
// 		this.permissions = key.Permissions;

// 		this.endpointPrefix = endpointPrefix;
// 	}

// 	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
// 		return this.api.userFromID(this.userID, options, noCache);
// 	}

// 	async get(): Promise<APIKey> {
// 		return new APIKey(
// 			this.api,
// 			await this.api.leashGet<LeashAPIKey>(`${this.endpointPrefix}`, {}, true),
// 			this.endpointPrefix
// 		);
// 	}

// 	async delete(): Promise<void> {
// 		await this.api.leashFetch(`${this.endpointPrefix}`, 'DELETE', undefined, true);
// 	}

// 	async update({ description, fullAccess, permissions }: APIKeyUpdateOptions): Promise<APIKey> {
// 		const updated = await this.api.leashFetch<LeashAPIKey>(`${this.endpointPrefix}`, 'PATCH', {
// 			description,
// 			full_access: fullAccess,
// 			permissions
// 		});

// 		return new APIKey(this.api, updated, this.endpointPrefix);
// 	}
// }

// interface LeashAPIKey {
// 	Key: string;
// 	CreatedAt: string;
// 	UpdatedAt: string;
// 	DeletedAt?: string;

// 	UserID: number;
// 	Description: string;
// 	FullAccess: boolean;
// 	Permissions: string[];
// }


use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};


#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiKey {
    #[serde(rename = "Key")]
    pub key: String,
    #[serde(rename = "CreatedAt")]
    pub created_at: DateTime<Utc>,
    #[serde(rename = "UpdatedAt")]
    pub updated_at: DateTime<Utc>,
    #[serde(rename = "DeletedAt")]
    pub deleted_at: Option<DateTime<Utc>>,

    #[serde(rename = "UserID")]
    pub user_id: u32,
    #[serde(rename = "Description")]
    pub description: String,
    #[serde(rename = "FullAccess")]
    pub full_access: bool,
    #[serde(rename = "Permissions")]
    pub permissions: Vec<String>,
}