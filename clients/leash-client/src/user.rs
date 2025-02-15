// export class User {
// 	private api: LeashAPI;

// 	id: number;
// 	createdAt: Date;
// 	updatedAt: Date;
// 	deletedAt?: Date;

// 	email: string;
// 	pendingEmail?: string;
// 	cardId: number;
// 	name: string;
// 	pronouns: string;
// 	role: string;
// 	type: string;

// 	// Student-like fields
// 	graduationYear: number;
// 	major: string;

// 	// Employee-like fields
// 	department: string;
// 	jobTitle: string;

// 	trainingsCache: ListAllCache<Training>;
// 	holdsCache: ListAllCache<Hold>;
// 	APIKeysCache: ListAllCache<APIKey>;
// 	userUpdatesCache: ListAllCache<UserUpdate>;
// 	notificationCache: ListAllCache<Notification>;

// 	permissions: string[];

// 	private endpointPrefix: string;

// 	constructor(
// 		api: LeashAPI,
// 		user: LeashUser,
// 		endpointPrefix: string,
// 		options: LeashUserOptions = {}
// 	) {
// 		this.api = api;
// 		this.id = user.ID;
// 		this.createdAt = new Date(user.CreatedAt);
// 		this.updatedAt = new Date(user.UpdatedAt);
// 		if (user.DeletedAt) {
// 			this.deletedAt = new Date(user.DeletedAt);
// 		}

// 		this.email = user.Email;
// 		this.pendingEmail = user.PendingEmail;
// 		this.cardId = user.CardID;
// 		this.name = user.Name;
// 		this.pronouns = user.Pronouns;
// 		this.role = user.Role;
// 		this.type = user.Type;

// 		// Student-like fields
// 		this.graduationYear = user.GraduationYear;
// 		this.major = user.Major;

// 		// Employee-like fields
// 		this.department = user.Department;
// 		this.jobTitle = user.JobTitle;

// 		this.permissions = user.Permissions;

// 		this.endpointPrefix = endpointPrefix;

// 		this.trainingsCache = new ListAllCache((includeDeleted: boolean, noCache: boolean) =>
// 			this.api.listAll(
// 				(options: LeashListOptions, noCache: boolean) => this.getTrainings(options, noCache),
// 				includeDeleted,
// 				100,
// 				noCache
// 			)
// 		);
// 		if (user.Trainings) {
// 			this.trainingsCache.setValue(
// 				user.Trainings.map(
// 					(training) =>
// 						new Training(api, training, `${this.endpointPrefix}/trainings/${training.Name}`)
// 				)
// 			);
// 		} else if (options.withTrainings) {
// 			this.trainingsCache.setValue([]);
// 		}

// 		this.holdsCache = new ListAllCache((includeDeleted: boolean, noCache: boolean) =>
// 			this.api.listAll(
// 				(options: LeashListOptions, noCache: boolean) => this.getHolds(options, noCache),
// 				includeDeleted,
// 				100,
// 				noCache
// 			)
// 		);
// 		if (user.Holds) {
// 			this.holdsCache.setValue(
// 				user.Holds.map(
// 					(hold) => new Hold(this.api, hold, `${this.endpointPrefix}/holds/${hold.Name}`)
// 				)
// 			);
// 		} else if (options.withHolds) {
// 			this.holdsCache.setValue([]);
// 		}

// 		this.APIKeysCache = new ListAllCache((includeDeleted: boolean, noCache: boolean) =>
// 			this.api.listAll(
// 				(options: LeashListOptions, noCache: boolean) => this.getAPIKeys(options, noCache),
// 				includeDeleted,
// 				100,
// 				noCache
// 			)
// 		);
// 		if (user.APIKeys) {
// 			this.APIKeysCache.setValue(
// 				user.APIKeys.map(
// 					(key) => new APIKey(this.api, key, `${this.endpointPrefix}/api_keys/${key.Key}`)
// 				)
// 			);
// 		} else if (options.withApiKeys) {
// 			this.APIKeysCache.setValue([]);
// 		}

// 		this.userUpdatesCache = new ListAllCache((includeDeleted: boolean, noCache: boolean) =>
// 			this.api.listAll(
// 				(options: LeashListOptions, noCache: boolean) => this.getUserUpdates(options, noCache),
// 				includeDeleted,
// 				100,
// 				noCache
// 			)
// 		);
// 		if (user.UserUpdates) {
// 			this.userUpdatesCache.setValue(
// 				user.UserUpdates.map((update) => new UserUpdate(this.api, update))
// 			);
// 		} else if (options.withUpdates) {
// 			this.userUpdatesCache.setValue([]);
// 		}

// 		this.notificationCache = new ListAllCache((includeDeleted: boolean, noCache: boolean) =>
// 			this.api.listAll(
// 				(options: LeashListOptions, noCache: boolean) => this.getNotifications(options, noCache),
// 				includeDeleted,
// 				100,
// 				noCache
// 			)
// 		);
// 		if (user.Notifications) {
// 			this.notificationCache.setValue(
// 				user.Notifications.map(
// 					(notification) =>
// 						new Notification(
// 							this.api,
// 							notification,
// 							`${this.endpointPrefix}/notifications/${notification.ID}`
// 						)
// 				)
// 			);
// 		} else {
// 			this.notificationCache.setValue([]);
// 		}
// 	}

// 	get iconURL(): string {
// 		if (!this.api.identiconURLCache.has(this.id)) {
// 			const svg = minidenticon(this.id.toString());
// 			const blob = new Blob([svg], { type: 'image/svg+xml' });
// 			this.api.identiconURLCache.set(this.id, URL.createObjectURL(blob));
// 		}

// 		return this.api.identiconURLCache.get(this.id) || '';
// 	}

// 	get roleNumber(): number {
// 		switch (this.role) {
// 			case 'service':
// 				return Role.USER_ROLE_SERVICE;
// 			case 'member':
// 				return Role.USER_ROLE_MEMBER;
// 			case 'volunteer':
// 				return Role.USER_ROLE_VOLUNTEER;
// 			case 'staff':
// 				return Role.USER_ROLE_STAFF;
// 			case 'admin':
// 				return Role.USER_ROLE_ADMIN;
// 			default:
// 				throw new Error(`Unknown role ${this.role}`);
// 		}
// 	}

// 	get isStaff(): boolean {
// 		return this.roleNumber >= Role.USER_ROLE_VOLUNTEER;
// 	}

// 	async getTrainings(
// 		options: LeashListOptions = {},
// 		noCache = false
// 	): Promise<LeashListResponse<Training>> {
// 		const prefix = `${this.endpointPrefix}/trainings`;
// 		const res = await this.api.leashList<LeashTraining, LeashListOptions>(prefix, options, noCache);
// 		return {
// 			total: res.total,
// 			data: res.data.map(
// 				(training) => new Training(this.api, training, `${prefix}/${training.Name}`)
// 			)
// 		};
// 	}

// 	async getAllTrainings(includeDeleted = false, noCache = false): Promise<Training[]> {
// 		return this.trainingsCache.get(includeDeleted, noCache);
// 	}

// 	async getHolds(
// 		options: LeashListOptions = {},
// 		noCache = false
// 	): Promise<LeashListResponse<Hold>> {
// 		const prefix = `${this.endpointPrefix}/holds`;
// 		const res = await this.api.leashList<LeashHold, LeashListOptions>(prefix, options, noCache);
// 		return {
// 			total: res.total,
// 			data: res.data.map((hold) => new Hold(this.api, hold, `${prefix}/${hold.Name}`))
// 		};
// 	}

// 	async getAllHolds(includeDeleted = false, noCache = false): Promise<Hold[]> {
// 		return this.holdsCache.get(includeDeleted, noCache);
// 	}

// 	async getAPIKeys(
// 		options: LeashListOptions = {},
// 		noCache = false
// 	): Promise<LeashListResponse<APIKey>> {
// 		const prefix = `${this.endpointPrefix}/apikeys`;
// 		const res = await this.api.leashList<LeashAPIKey, LeashListOptions>(prefix, options, noCache);
// 		return {
// 			total: res.total,
// 			data: res.data.map((key) => new APIKey(this.api, key, `${prefix}/${key.Key}`))
// 		};
// 	}

// 	async getAllAPIKeys(includeDeleted = false, noCache = false): Promise<APIKey[]> {
// 		return this.APIKeysCache.get(includeDeleted, noCache);
// 	}

// 	async getUserUpdates(
// 		options: LeashListOptions = {},
// 		noCache = false
// 	): Promise<LeashListResponse<UserUpdate>> {
// 		const res = await this.api.leashList<LeashUserUpdate, LeashListOptions>(
// 			`${this.endpointPrefix}/updates`,
// 			options,
// 			noCache
// 		);
// 		return {
// 			total: res.total,
// 			data: res.data.map((update) => new UserUpdate(this.api, update))
// 		};
// 	}

// 	async getAllUserUpdates(includeDeleted = false, noCache = false): Promise<UserUpdate[]> {
// 		return this.userUpdatesCache.get(includeDeleted, noCache);
// 	}

// 	async getNotifications(
// 		options: LeashListOptions = {},
// 		noCache = false
// 	): Promise<LeashListResponse<Notification>> {
// 		const prefix = `${this.endpointPrefix}/notifications`;
// 		const res = await this.api.leashList<LeashNotification, LeashListOptions>(
// 			prefix,
// 			options,
// 			noCache
// 		);
// 		return {
// 			total: res.total,
// 			data: res.data.map(
// 				(notification) => new Notification(this.api, notification, `${prefix}/${notification.ID}`)
// 			)
// 		};
// 	}

// 	async getAllNotifications(includeDeleted = false, noCache = false): Promise<Notification[]> {
// 		return this.notificationCache.get(includeDeleted, noCache);
// 	}

// 	async get(options: LeashUserOptions = {}): Promise<User> {
// 		return new User(
// 			this.api,
// 			await this.api.leashGet<LeashUser>(`${this.endpointPrefix}`, options, true),
// 			this.endpointPrefix,
// 			options
// 		);
// 	}

// 	async delete(): Promise<void> {
// 		await this.api.leashFetch(`${this.endpointPrefix}`, 'DELETE', undefined, true);
// 	}

// 	async update({
// 		name,
// 		pronouns,
// 		email,
// 		cardID,
// 		role,
// 		type,
// 		graduationYear,
// 		major,
// 		department,
// 		jobTitle
// 	}: UserUpdateOptions): Promise<User> {
// 		if (this.role === 'service') {
// 			throw new Error('Service users cannot be updated with this method.');
// 		}

// 		const updated = await this.api.leashFetch<LeashUser>(`${this.endpointPrefix}`, 'PATCH', {
// 			name,
// 			pronouns,
// 			email,
// 			card_id: cardID,
// 			role,
// 			type,
// 			graduation_year: graduationYear,
// 			major,
// 			department,
// 			job_title: jobTitle
// 		});

// 		return new User(this.api, updated, this.endpointPrefix);
// 	}

// 	async updateService({ name, permissions }: ServiceUserUpdateOptions): Promise<User> {
// 		if (this.role !== 'service') {
// 			throw new Error('Only service users can be updated with this method.');
// 		}

// 		const updated = await this.api.leashFetch<LeashUser>(
// 			`${this.endpointPrefix}/service`,
// 			'PATCH',
// 			{
// 				name,
// 				permissions
// 			}
// 		);

// 		return new User(this.api, updated, this.endpointPrefix);
// 	}

// 	async createAPIKey({
// 		description,
// 		fullAccess,
// 		permissions
// 	}: APIKeyCreateOptions): Promise<APIKey> {
// 		const key = await this.api.leashFetch<LeashAPIKey>(`${this.endpointPrefix}/apikeys`, 'POST', {
// 			description,
// 			full_access: fullAccess,
// 			permissions
// 		});

// 		this.APIKeysCache.invalidate();

// 		return new APIKey(this.api, key, `${this.endpointPrefix}/apikeys/${key.Key}`);
// 	}

// 	async getAPIKey(key: string, noCache = false): Promise<APIKey> {
// 		return new APIKey(
// 			this.api,
// 			await this.api.leashGet<LeashAPIKey>(`${this.endpointPrefix}/apikeys/${key}`, {}, noCache),
// 			`${this.endpointPrefix}/apikeys/${key}`
// 		);
// 	}

// 	async createTraining({ name, level }: TrainingCreateOptions): Promise<Training> {
// 		const training = await this.api.leashFetch<LeashTraining>(
// 			`${this.endpointPrefix}/trainings`,
// 			'POST',
// 			{
// 				name,
// 				level
// 			}
// 		);

// 		this.trainingsCache.invalidate();

// 		return new Training(this.api, training, `${this.endpointPrefix}/trainings/${training.Name}`);
// 	}

// 	async getTraining(name: string, noCache = false): Promise<Training> {
// 		return new Training(
// 			this.api,
// 			await this.api.leashGet<LeashTraining>(
// 				`${this.endpointPrefix}/trainings/${name}`,
// 				{},
// 				noCache
// 			),
// 			`${this.endpointPrefix}/trainings/${name}`
// 		);
// 	}

// 	async createHold({
// 		name,
// 		reason,
// 		start,
// 		end,
// 		priority,
// 		resolutionLink
// 	}: HoldCreateOptions): Promise<Hold> {
// 		const hold = await this.api.leashFetch<LeashHold>(`${this.endpointPrefix}/holds`, 'POST', {
// 			name,
// 			reason,
// 			start,
// 			end,
// 			priority,
// 			resolution_link: resolutionLink
// 		});

// 		this.holdsCache.invalidate();

// 		return new Hold(this.api, hold, `${this.endpointPrefix}/holds/${hold.Name}`);
// 	}

// 	async getHold(name: string, noCache = false): Promise<Hold> {
// 		return new Hold(
// 			this.api,
// 			await this.api.leashGet<LeashHold>(`${this.endpointPrefix}/holds/${name}`, {}, noCache),
// 			`${this.endpointPrefix}/holds/${name}`
// 		);
// 	}

// 	async createNotification({
// 		title,
// 		message,
// 		link,
// 		group
// 	}: NotificationCreateOptions): Promise<Notification> {
// 		const notification = await this.api.leashFetch<LeashNotification>(
// 			`${this.endpointPrefix}/notifications`,
// 			'POST',
// 			{
// 				title,
// 				message,
// 				link,
// 				group
// 			}
// 		);

// 		this.notificationCache.invalidate();

// 		return new Notification(
// 			this.api,
// 			notification,
// 			`${this.endpointPrefix}/notifications/${notification.ID}`
// 		);
// 	}

// 	async getNotification(notificationID: number, noCache = false): Promise<Notification> {
// 		return new Notification(
// 			this.api,
// 			await this.api.leashGet<LeashNotification>(
// 				`${this.endpointPrefix}/notifications/${notificationID}`,
// 				{},
// 				noCache
// 			),
// 			`${this.endpointPrefix}/notifications/${notificationID}`
// 		);
// 	}

// 	async getAllPermissions(noCache = false): Promise<string[]> {
// 		return await this.api.leashGet<string[]>(`${this.endpointPrefix}/permissions`, {}, noCache);
// 	}

// 	async checkin(): Promise<CheckinResponse> {
// 		const res = await this.api.leashGet<LeashCheckinResponse>(
// 			`${this.endpointPrefix}/checkin`,
// 			undefined,
// 			true
// 		);
// 		return {
// 			token: res.token,
// 			expiresAt: new Date(res.expires_at)
// 		};
// 	}
// }

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

use crate::{apikey::ApiKey, hold::Hold, notification::Notification, training::Training};

// interface LeashUser {
// 	ID: number;
// 	CreatedAt: string;
// 	UpdatedAt: string;
// 	DeletedAt?: string;

// 	Email: string;
// 	PendingEmail?: string;
// 	CardID: string;
// 	Name: string;
// 	Pronouns: string;
// 	Role: string;
// 	Type: string;

// 	// Student-like fields
// 	GraduationYear: number;
// 	Major: string;

// 	// Employee-like fields
// 	Department: string;
// 	JobTitle: string;

// 	Trainings?: LeashTraining[];
// 	Holds?: LeashHold[];
// 	APIKeys?: LeashAPIKey[];
// 	UserUpdates?: LeashUserUpdate[];
// 	Notifications?: LeashNotification[];

// 	Permissions: string[];
// }

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    #[serde(rename = "ID")]
    pub id: u32,
    #[serde(rename = "CreatedAt")]
    pub created_at: DateTime<Utc>,
    #[serde(rename = "UpdatedAt")]
    pub updated_at: DateTime<Utc>,
    #[serde(rename = "DeletedAt")]
    pub deleted_at: Option<DateTime<Utc>>,

    #[serde(rename = "Email")]
    pub email: String,
    #[serde(rename = "PendingEmail")]
    pub pending_email: Option<String>,
    #[serde(rename = "CardID")]
    pub card_id: Option<String>,
    #[serde(rename = "Name")]
    pub name: String,
    #[serde(rename = "Pronouns")]
    pub pronouns: String,
    #[serde(rename = "Role")]
    pub role: String,
    #[serde(rename = "Type")]
    pub user_type: String,

    // Student-like fields
    #[serde(rename = "GraduationYear")]
    pub graduation_year: Option<u32>,
    #[serde(rename = "Major")]
    pub major: Option<String>,

    // Employee-like fields
    #[serde(rename = "Department")]
    pub department: Option<String>,
    #[serde(rename = "JobTitle")]
    pub job_title: Option<String>,

    #[serde(rename = "Trainings")]
    preload_trainings: Option<Vec<Training>>,
    #[serde(rename = "Holds")]
    preload_holds: Option<Vec<Hold>>,
    #[serde(rename = "APIKeys")]
    preload_api_keys: Option<Vec<ApiKey>>,
    #[serde(rename = "UserUpdates")]
    preload_user_updates: Option<Vec<UserUpdate>>,
    #[serde(rename = "Notifications")]
    preload_notifications: Option<Vec<Notification>>,

    #[serde(rename = "Permissions")]
    pub permissions: Vec<String>,

    #[serde(skip)]
    pub endpoint_prefix: String,
}

// interface LeashUserUpdate {
// 	ID: number;
// 	CreatedAt: string;
// 	UpdatedAt: string;
// 	DeletedAt?: string;

// 	UserID: number;
// 	EditedBy: number;
// 	Field: string;
// 	OldValue: string;
// 	NewValue: string;
// }

// export class UserUpdate {
// 	private api: LeashAPI;
// 	id: number;
// 	createdAt: Date;
// 	updatedAt: Date;
// 	deletedAt?: Date;

// 	private userID: number;
// 	private editedById: number;

// 	field: string;
// 	oldValue: string;
// 	newValue: string;

// 	constructor(api: LeashAPI, update: LeashUserUpdate) {
// 		this.api = api;
// 		this.id = update.ID;
// 		this.createdAt = new Date(update.CreatedAt);
// 		this.updatedAt = new Date(update.UpdatedAt);
// 		if (update.DeletedAt) {
// 			this.deletedAt = new Date(update.DeletedAt);
// 		}

// 		this.userID = update.UserID;
// 		this.editedById = update.EditedBy;

// 		this.field = update.Field;
// 		this.oldValue = update.OldValue;
// 		this.newValue = update.NewValue;
// 	}

// 	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
// 		return this.api.userFromID(this.userID, options, noCache);
// 	}

// 	async getEditedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
// 		return this.api.userFromID(this.editedById, options, noCache);
// 	}
// }

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserUpdate {
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
    #[serde(rename = "EditedBy")]
    pub edited_by: u32,
    #[serde(rename = "Field")]
    pub field: String,
    #[serde(rename = "OldValue")]
    pub old_value: String,
    #[serde(rename = "NewValue")]
    pub new_value: String,
}


// interface LeashTokenRefresh {
// 	token: string;
// 	expires_at: string;
// }