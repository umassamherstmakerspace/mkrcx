// interface LeashTraining {
// 	ID: number;
// 	CreatedAt: string;
// 	UpdatedAt: string;
// 	DeletedAt?: string;

// 	UserID: number;
// 	Name: string;
// 	Level: TrainingLevel;
// 	AddedBy: number;
// 	RemovedBy?: number;
// }


// export class Training {
// 	private api: LeashAPI;
// 	id: number;
// 	createdAt: Date;
// 	updatedAt: Date;
// 	deletedAt?: Date;

// 	name: string;
// 	level: TrainingLevel;

// 	private userID: number;
// 	private addedById: number;
// 	private removedById?: number;

// 	private endpointPrefix: string;

// 	constructor(api: LeashAPI, training: LeashTraining, endpointPrefix: string) {
// 		this.api = api;
// 		this.id = training.ID;
// 		this.createdAt = new Date(training.CreatedAt);
// 		this.updatedAt = new Date(training.UpdatedAt);
// 		if (training.DeletedAt) {
// 			this.deletedAt = new Date(training.DeletedAt);
// 		}

// 		this.name = training.Name;
// 		this.level = training.Level;
// 		this.userID = training.UserID;
// 		this.addedById = training.AddedBy;
// 		if (training.RemovedBy) {
// 			this.removedById = training.RemovedBy;
// 		}

// 		this.endpointPrefix = endpointPrefix;
// 	}

// 	levelString(): string {
// 		return trainingLevelToString(this.level);
// 	}

// 	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
// 		return this.api.userFromID(this.userID, options, noCache);
// 	}

// 	async getAddedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
// 		return this.api.userFromID(this.addedById, options, noCache);
// 	}

// 	async getRemovedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
// 		if (!this.removedById) {
// 			throw new Error('Training has not been removed.');
// 		}

// 		return this.api.userFromID(this.removedById, options, noCache);
// 	}

// 	async get(): Promise<Training> {
// 		return new Training(
// 			this.api,
// 			await this.api.leashGet<LeashTraining>(`${this.endpointPrefix}`, {}, true),
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
pub struct Training {
    #[serde(rename = "ID")]
    pub id: u32,
    #[serde(rename = "CreatedAt")]
    pub created_at: DateTime<Utc>,
    #[serde(rename = "UpdatedAt")]
    pub updated_at: DateTime<Utc>,
    #[serde(rename = "DeletedAt")]
    pub deleted_at: Option<DateTime<Utc>>,

    pub name: String,
    // pub level: TrainingLevel,

    #[serde(rename = "UserID")]
    pub user_id: u32,
    #[serde(rename = "AddedBy")]
    pub added_by: u32,
    #[serde(rename = "RemovedBy")]
    pub removed_by: Option<u32>,

    pub endpoint_prefix: String,
}