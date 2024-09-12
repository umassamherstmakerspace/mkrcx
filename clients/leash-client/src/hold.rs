// interface LeashHold {
// 	ID: number;
// 	CreatedAt: string;
// 	UpdatedAt: string;
// 	DeletedAt?: string;

// 	UserID: number;
// 	Name: string;
// 	Reason: string;
// 	Start?: string;
// 	End?: string;
// 	ResolutionLink?: string;
// 	AddedBy: number;
// 	RemovedBy?: number;

// 	Priority: number;
// }


// export class Hold {
// 	private api: LeashAPI;
// 	id: number;
// 	createdAt: Date;
// 	updatedAt: Date;
// 	deletedAt?: Date;

// 	name: string;
// 	reason: string;
// 	start?: Date;
// 	end?: Date;

// 	resolutionLink?: string;

// 	priority: number;

// 	private userID: number;
// 	private addedById: number;
// 	private removedById?: number;

// 	private endpointPrefix: string;

// 	constructor(api: LeashAPI, hold: LeashHold, endpointPrefix: string) {
// 		this.api = api;
// 		this.id = hold.ID;
// 		this.createdAt = new Date(hold.CreatedAt);
// 		this.updatedAt = new Date(hold.UpdatedAt);
// 		if (hold.DeletedAt) {
// 			this.deletedAt = new Date(hold.DeletedAt);
// 		}

// 		this.name = hold.Name;
// 		this.reason = hold.Reason;
// 		if (hold.Start) {
// 			this.start = new Date(hold.Start);
// 		}
// 		if (hold.End) {
// 			this.end = new Date(hold.End);
// 		}

// 		this.userID = hold.UserID;
// 		this.addedById = hold.AddedBy;
// 		if (hold.RemovedBy) {
// 			this.removedById = hold.RemovedBy;
// 		}

// 		this.resolutionLink = hold.ResolutionLink;

// 		this.priority = hold.Priority;

// 		this.endpointPrefix = endpointPrefix;

// 		if (this.end && !this.deletedAt && !this.removedById && isAfter(new Date(), this.end)) {
// 			this.deletedAt = this.end;
// 			this.removedById = this.addedById;
// 		}
// 	}

// 	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
// 		return this.api.userFromID(this.userID, options, noCache);
// 	}

// 	async getAddedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
// 		return this.api.userFromID(this.addedById, options, noCache);
// 	}

// 	async getRemovedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
// 		if (!this.removedById) {
// 			throw new Error('Hold has not been removed.');
// 		}

// 		return this.api.userFromID(this.removedById, options, noCache);
// 	}

// 	isPending(): boolean {
// 		const ended = this.end ? isAfter(new Date(), this.end) : false;
// 		const started = this.start ? isAfter(new Date(), this.start) : true;
// 		return !started && !ended && this.deletedAt === undefined;
// 	}

// 	isActive(): boolean {
// 		const ended = this.end ? isAfter(new Date(), this.end) : false;
// 		const started = this.start ? isAfter(new Date(), this.start) : true;
// 		return started && !ended && this.deletedAt === undefined;
// 	}

// 	async get(): Promise<Hold> {
// 		return new Hold(
// 			this.api,
// 			await this.api.leashGet<LeashHold>(`${this.endpointPrefix}`, {}, true),
// 			this.endpointPrefix
// 		);
// 	}

// 	async delete(): Promise<void> {
// 		await this.api.leashFetch(`${this.endpointPrefix}`, 'DELETE', undefined, true);
// 	}

// 	activeLevel(): number {
// 		if (this.isActive()) {
// 			return 0;
// 		} else if (this.isPending()) {
// 			return 1;
// 		} else {
// 			return 2;
// 		}
// 	}
// }

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};


#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Hold {
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
    #[serde(rename = "Name")]
    pub name: String,
    #[serde(rename = "Reason")]
    pub reason: String,
    #[serde(rename = "Start")]
    pub start: Option<DateTime<Utc>>,
    #[serde(rename = "End")]
    pub end: Option<DateTime<Utc>>,
    #[serde(rename = "ResolutionLink")]
    pub resolution_link: Option<String>,
    #[serde(rename = "AddedBy")]
    pub added_by: u32,
    #[serde(rename = "RemovedBy")]
    pub removed_by: Option<u32>,

    #[serde(rename = "Priority")]
    pub priority: u32,
}