// interface LeashNotification {
// 	ID: number;
// 	CreatedAt: string;
// 	UpdatedAt: string;
// 	DeletedAt?: string;

// 	UserID: number;
// 	Title: string;
// 	Message: string;
// 	Link: string;
// 	Group: string;

// 	AddedBy: number;
// }

// export class Notification {
// 	private api: LeashAPI;
// 	id: number;
// 	createdAt: Date;
// 	updatedAt: Date;
// 	deletedAt?: Date;

// 	private userID: number;

// 	title: string;
// 	message: string;
// 	link: string;
// 	group: string;

// 	private addedById: number;

// 	private endpointPrefix: string;

// 	constructor(api: LeashAPI, notification: LeashNotification, endpointPrefix: string) {
// 		this.api = api;
// 		this.id = notification.ID;
// 		this.createdAt = new Date(notification.CreatedAt);
// 		this.updatedAt = new Date(notification.UpdatedAt);
// 		if (notification.DeletedAt) {
// 			this.deletedAt = new Date(notification.DeletedAt);
// 		}

// 		this.userID = notification.UserID;

// 		this.title = notification.Title;
// 		this.message = notification.Message;
// 		this.link = notification.Link;
// 		this.group = notification.Group;

// 		this.addedById = notification.AddedBy;

// 		this.endpointPrefix = endpointPrefix;
// 	}

// 	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
// 		return this.api.userFromID(this.userID, options, noCache);
// 	}

// 	async getAddedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
// 		return this.api.userFromID(this.addedById, options, noCache);
// 	}

// 	async get(): Promise<Notification> {
// 		return new Notification(
// 			this.api,
// 			await this.api.leashGet<LeashNotification>(`${this.endpointPrefix}`, {}, true),
// 			this.endpointPrefix
// 		);
// 	}

// 	async delete(): Promise<void> {
// 		await this.api.leashFetch(`${this.endpointPrefix}`, 'DELETE', undefined, true);
// 	}
// }


use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};


#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Notification {
    #[serde(rename = "ID")]
    pub id: u32,
    #[serde(rename = "CreatedAt")]
    pub created_at: DateTime<Utc>,
    #[serde(rename = "UpdatedAt")]
    pub updated_at: DateTime<Utc>,
    #[serde(rename = "DeletedAt")]
    pub deleted_at: Option<DateTime<Utc>>,

    #[serde(rename = "UserID")]
    pub user_id: u32,

    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "Message")]
    pub message: String,
    #[serde(rename = "Link")]
    pub link: String,
    #[serde(rename = "Group")]
    pub group: String,

    #[serde(rename = "AddedBy")]
    pub added_by: u32,

    #[serde(skip)]
    pub endpoint_prefix: String,
}
