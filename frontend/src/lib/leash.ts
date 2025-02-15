import { Cached } from './cache';
import { isAfter } from 'date-fns';
import { minidenticon } from 'minidenticons';

export const allPermissions = [
	'leash.users:target_self',
	'leash.users:target_others',
	'leash.users:create',
	'leash.users.service:create',
	'leash.users:search',
	'leash.users.get:email',
	'leash.users.get:card',
	'leash.users.get:checkin',
	'leash.users.get.trainings:list',
	'leash.users.get.holds:list',
	'leash.users.get.apikeys:list',
	'leash.users.get.updates:list',
	'leash.users.get.notifications:list',
	'leash.users.self:get',
	'leash.users.self:update',
	'leash.users.self:update_card_id',
	'leash.users.self:update_role',
	'leash.users.self:service_update',
	'leash.users.self:checkin',
	'leash.users.self:permissions',
	'leash.users.self.updates:list',
	'leash.users.self.trainings:target',
	'leash.users.self.trainings:list',
	'leash.users.self.trainings:get',
	'leash.users.self.trainings:create',
	'leash.users.self.trainings:delete',
	'leash.users.self.holds:target',
	'leash.users.self.holds:list',
	'leash.users.self.holds:create',
	'leash.users.self.holds:get',
	'leash.users.self.holds:delete',
	'leash.users.self.apikeys:target',
	'leash.users.self.apikeys:list',
	'leash.users.self.apikeys:create',
	'leash.users.self.apikeys:get',
	'leash.users.self.apikeys:update',
	'leash.users.self.apikeys:delete',
	'leash.users.self.notifications:target',
	'leash.users.self.notifications:list',
	'leash.users.self.notifications:get',
	'leash.users.self.notifications:delete',
	'leash.users.self.notifications:create',
	'leash.users.others:get',
	'leash.users.others:update',
	'leash.users.others:update_card_id',
	'leash.users.others:update_role',
	'leash.users.others:service_update',
	'leash.users.others:delete',
	'leash.users.others:checkin',
	'leash.users.others:permissions',
	'leash.users.others.updates:list',
	'leash.users.others.trainings:target',
	'leash.users.others.trainings:list',
	'leash.users.others.trainings:get',
	'leash.users.others.trainings:create',
	'leash.users.others.trainings:delete',
	'leash.users.others.holds:target',
	'leash.users.others.holds:list',
	'leash.users.others.holds:create',
	'leash.users.others.holds:get',
	'leash.users.others.holds:delete',
	'leash.users.others.apikeys:target',
	'leash.users.others.apikeys:list',
	'leash.users.others.apikeys:create',
	'leash.users.others.apikeys:get',
	'leash.users.others.apikeys:delete',
	'leash.users.others.apikeys:update',
	'leash.users.others.notifications:target',
	'leash.users.others.notifications:list',
	'leash.users.others.notifications:get',
	'leash.users.others.notifications:delete',
	'leash.users.others.notifications:create',
	'leash.trainings:target',
	'leash.trainings:get',
	'leash.trainings:delete',
	'leash.holds:target',
	'leash.holds:get',
	'leash.holds:delete',
	'leash.apikeys:target',
	'leash.apikeys:get',
	'leash.apikeys:delete',
	'leash.apikeys:update',
	'leash.notifications:get',
	'leash.notifications:delete'
];

export const permissionOptions = allPermissions.map((permission) => ({
	name: permission,
	value: permission
}));

export enum Role {
	USER_ROLE_SERVICE = 0,
	USER_ROLE_MEMBER = 1,
	USER_ROLE_VOLUNTEER = 2,
	USER_ROLE_STAFF = 3,
	USER_ROLE_ADMIN = 4
}

interface LeashUser {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	DeletedAt?: string;

	Email: string;
	PendingEmail?: string;
	CardID: string;
	Name: string;
	Pronouns: string;
	Role: string;
	Type: string;

	// Student-like fields
	GraduationYear: number;
	Major: string;

	// Employee-like fields
	Department: string;
	JobTitle: string;

	Trainings?: LeashTraining[];
	Holds?: LeashHold[];
	APIKeys?: LeashAPIKey[];
	UserUpdates?: LeashUserUpdate[];
	Notifications?: LeashNotification[];

	Permissions: string[];
}

interface LeashAPIKey {
	Key: string;
	CreatedAt: string;
	UpdatedAt: string;
	DeletedAt?: string;

	UserID: number;
	Description: string;
	FullAccess: boolean;
	Permissions: string[];
}
export const enum TrainingLevel {
	IN_PROGRESS = 'in_progress',
	SUPERVISED = 'supervised',
	UNSUPERVISED = 'unsupervised',
	CAN_TRAIN = 'can_train'
}

export function trainingLevelToString(level: TrainingLevel): string {
	switch (level) {
		case TrainingLevel.IN_PROGRESS:
			return 'In Progress';
		case TrainingLevel.SUPERVISED:
			return 'Requires Active Staff Supervision';
		case TrainingLevel.UNSUPERVISED:
			return 'Does Not Require Active Staff Supervision';
		case TrainingLevel.CAN_TRAIN:
			return 'Can Train Others';
	}
}

interface LeashTraining {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	DeletedAt?: string;

	UserID: number;
	Name: string;
	Level: TrainingLevel;
	AddedBy: number;
	RemovedBy?: number;
}

interface LeashHold {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	DeletedAt?: string;

	UserID: number;
	Name: string;
	Reason: string;
	Start?: string;
	End?: string;
	ResolutionLink?: string;
	AddedBy: number;
	RemovedBy?: number;

	Priority: number;
}

interface LeashUserUpdate {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	DeletedAt?: string;

	UserID: number;
	EditedBy: number;
	Field: string;
	OldValue: string;
	NewValue: string;
}

interface LeashNotification {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	DeletedAt?: string;

	UserID: number;
	Title: string;
	Message: string;
	Link: string;
	Group: string;

	AddedBy: number;
}

interface LeashTokenRefresh {
	token: string;
	expires_at: string;
}

export interface UserCreateOptions {
	email: string;
	name: string;
	pronouns: string;
	role: string;
	type: string;

	// Student-like fields
	graduationYear?: number;
	major?: string;

	// Employee-like fields
	department?: string;
	jobTitle?: string;
}

export interface UserUpdateOptions {
	name?: string;
	pronouns?: string;
	email?: string;
	cardID?: string;
	role?: string;
	type?: string;

	// Student-like fields
	graduationYear?: number;
	major?: string;

	// Employee-like fields
	department?: string;
	jobTitle?: string;
}

export interface ServiceUserCreateOptions {
	name: string;
	permissions: string[];
}

export interface ServiceUserUpdateOptions {
	name?: string;
	permissions?: string[];
}

export interface APIKeyCreateOptions {
	description?: string;
	fullAccess?: boolean;
	permissions?: string[];
}

export interface APIKeyUpdateOptions {
	description?: string;
	fullAccess?: boolean;
	permissions?: string[];
}

export interface TrainingCreateOptions {
	name: string;
	level: TrainingLevel;
}

export interface HoldCreateOptions {
	name: string;
	reason: string;
	start?: number;
	end?: number;
	priority: number;
	resolutionLink?: string;
}

export interface NotificationCreateOptions {
	title: string;
	message: string;
	link?: string;
	group?: string;
}

export interface LeashListResponse<T> {
	total: number;
	data: T[];
}

interface LeashCheckinResponse {
	token: string;
	expires_at: string;
}

interface CheckinResponse {
	token: string;
	expiresAt: Date;
}

export interface LeashUserOptions {
	withTrainings?: boolean;
	withHolds?: boolean;
	withApiKeys?: boolean;
	withUpdates?: boolean;
	withNotifications?: boolean;
}

export interface LeashListOptions {
	offset?: number;
	limit?: number;
	includeDeleted?: boolean;
}

export class ListAllCache<T> {
	private getter: (includeDeleted: boolean, noCache: boolean) => Promise<T[]>;
	private cache: Cached<T[]>;
	private deletedCache: Cached<T[]>;

	constructor(getter: (includeDeleted: boolean, noCache: boolean) => Promise<T[]>) {
		this.getter = getter;
		this.cache = new Cached(() => getter(false, false));
		this.deletedCache = new Cached(() => getter(true, false));
	}

	public setValue(val: T[]) {
		this.cache.setValue(val);
	}

	public invalidate() {
		this.cache.invalidate();
		this.deletedCache.invalidate();
	}

	public async get(includeDeleted = false, noCache = false): Promise<T[]> {
		if (noCache) {
			return await this.getter(includeDeleted, noCache);
		}

		const cache = includeDeleted ? this.deletedCache : this.cache;
		return await cache.get();
	}
}

export interface LeashUserSearchOptions extends LeashListOptions, LeashUserOptions {
	showService?: boolean;
}

type LeashListGetter<T> = (
	options: LeashListOptions,
	nonCahce: boolean
) => Promise<LeashListResponse<T>>;

const camelToSnakeCase: (str: string) => string = (str) =>
	str.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);

export class LeashAPI {
	private token: string;
	private leashURL: string;
	private fetchFunction: typeof fetch = fetch;
	public identiconURLCache = new Map<number, string>();
	private getCache = new Map<string, Cached<object>>();

	constructor(token: string, leashURL: string) {
		this.token = token;
		this.leashURL = leashURL;
	}

	public overrideFetchFunction(fetchFunction: typeof fetch): void {
		this.fetchFunction = fetchFunction;
	}

	async leashFetch<T extends object>(
		endpoint: string,
		method: string,
		body?: object,
		noResponse = false
	): Promise<T> {
		const r = await this.fetchFunction(`${this.leashURL}${endpoint}`, {
			method: method,
			headers: {
				Authorization: `Bearer ${this.token}`,
				'Content-Type': 'application/json'
			},
			redirect: 'follow',
			mode: 'cors',
			cache: 'no-cache',
			credentials: 'same-origin',
			body: JSON.stringify(body)
		});
		return await (Math.floor(r.status / 100) !== 2
			? Promise.reject(new Error(await r.text()))
			: noResponse
				? r.text()
				: r.json());
	}

	async leashGet<T extends object>(
		endpoint: string,
		options: object = {},
		noCache = false
	): Promise<T> {
		let args = Object.entries(options)
			.map(([key, value]) => `${camelToSnakeCase(key)}=${value}`)
			.join('&');
		if (args.length > 0) {
			args = `?${args}`;
		}

		const link = `${endpoint}${args}`;

		if (noCache) {
			return this.leashFetch<T>(link, 'GET');
		} else {
			if (!this.getCache.has(link)) {
				this.getCache.set(link, new Cached(() => this.leashFetch<T>(link, 'GET')));
			}

			return this.getCache.get(link)?.get() as T;
		}
	}

	async leashList<T, O extends LeashListOptions>(
		endpoint: string,
		options: O | Record<string, never> = {},
		noCache = false
	): Promise<LeashListResponse<T>> {
		return this.leashGet<LeashListResponse<T>>(endpoint, options, noCache);
	}

	async listAll<T>(
		getter: LeashListGetter<T>,
		includeDeleted = false,
		limit = 100,
		noCache = false
	): Promise<T[]> {
		let offset = 0;
		let result: T[] = [];
		let currentResult: LeashListResponse<T>;
		do {
			currentResult = await getter(
				{
					offset,
					limit,
					includeDeleted
				},
				noCache
			);
			result = result.concat(currentResult.data);
			offset += limit;
		} while (currentResult.total > offset);

		return result;
	}

	public async createUser({
		email,
		name,
		pronouns,
		role,
		type,
		graduationYear,
		major,
		department,
		jobTitle
	}: UserCreateOptions): Promise<User> {
		const user = await this.leashFetch<LeashUser>(`/api/users`, 'POST', {
			email,
			name,
			pronouns,
			role,
			type,
			graduation_year: graduationYear,
			major,
			department,
			job_title: jobTitle
		});

		return new User(this, user, `/api/users/${user.ID}`);
	}

	public async createServiceUser({ name, permissions }: ServiceUserCreateOptions): Promise<User> {
		const user = await this.leashFetch<LeashUser>(`/api/users/service`, 'POST', {
			name,
			permissions
		});

		return new User(this, user, `/api/users/${user.ID}`);
	}

	public async searchUsers(
		query: string,
		options: LeashUserSearchOptions = {},
		noCache = false
	): Promise<LeashListResponse<User>> {
		interface LeashUserSearchOptionsWhole extends LeashUserSearchOptions {
			query: string;
		}

		const optionsWhole: LeashUserSearchOptionsWhole = {
			...options,
			query
		};

		const res = await this.leashList<LeashUser, LeashUserSearchOptionsWhole>(
			`/api/users/search/`,
			optionsWhole,
			noCache
		);

		return {
			total: res.total,
			data: res.data.map((user) => new User(this, user, `/api/users/${user.ID}`, options))
		};
	}

	public async userFromID(
		id: number,
		options: LeashUserOptions = {},
		noCache = false
	): Promise<User> {
		return this.leashGet<LeashUser>(`/api/users/${id}`, options, noCache).then(
			(user) => new User(this, user, `/api/users/${id}`, options)
		);
	}

	public async selfUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.leashGet<LeashUser>(`/api/users/self`, options, noCache).then(
			(user) => new User(this, user, `/api/users/self`, options)
		);
	}

	public async userFromEmail(
		email: string,
		options: LeashUserOptions = {},
		noCache = false
	): Promise<User> {
		return this.leashGet<LeashUser>(`/api/users/get/email/${email}`, options, noCache).then(
			(user) => new User(this, user, `/api/users/${user.ID}`, options)
		);
	}

	public async userFromCardID(
		cardID: string,
		options: LeashUserOptions = {},
		noCache = false
	): Promise<User> {
		return this.leashGet<LeashUser>(`/api/users/get/card/${cardID}`, options, noCache).then(
			(user) => new User(this, user, `/api/users/${user.ID}`, options)
		);
	}

	public async apiKeyFromKey(key: string, noCache = false): Promise<APIKey> {
		return this.leashGet<LeashAPIKey>(`/api/apikeys/${key}`, {}, noCache).then(
			(key) => new APIKey(this, key, `/api/apikeys/${key.Key}`)
		);
	}

	public async trainingFromID(id: number, noCache = false): Promise<Training> {
		return this.leashGet<LeashTraining>(`/api/trainings/${id}`, {}, noCache).then(
			(training) => new Training(this, training, `/api/trainings/${id}`)
		);
	}

	public async holdFromID(id: number, noCache = false): Promise<Hold> {
		return this.leashGet<LeashHold>(`/api/holds/${id}`, {}, noCache).then(
			(hold) => new Hold(this, hold, `/api/holds/${id}`)
		);
	}

	public async notificationFromID(id: number, noCache = false): Promise<Notification> {
		return this.leashGet<LeashNotification>(`/api/notifications/${id}`, {}, noCache).then(
			(notification) => new Notification(this, notification, `/api/notifications/${id}`)
		);
	}

	public async refreshTokens(): Promise<LeashTokenRefresh> {
		return this.leashFetch<LeashTokenRefresh>(`/auth/refresh`, 'GET');
	}

	public async validateToken(): Promise<boolean> {
		try {
			await this.leashFetch(`/auth/validate`, 'GET', undefined, true);
			return true;
		} catch (e) {
			return false;
		}
	}

	public login(login: string, return_to: string): string {
		const state = btoa(return_to);

		return `${this.leashURL}/auth/login?return=${login}&state=${state}`;
	}

	public logout(return_to: string): string {
		return `${this.leashURL}/auth/logout?token=${this.token}&return=${return_to}`;
	}
}

export class User {
	private api: LeashAPI;

	id: number;
	createdAt: Date;
	updatedAt: Date;
	deletedAt?: Date;

	email: string;
	pendingEmail?: string;
	cardId: string;
	name: string;
	pronouns: string;
	role: string;
	type: string;

	// Student-like fields
	graduationYear: number;
	major: string;

	// Employee-like fields
	department: string;
	jobTitle: string;

	trainingsCache: ListAllCache<Training>;
	holdsCache: ListAllCache<Hold>;
	APIKeysCache: ListAllCache<APIKey>;
	userUpdatesCache: ListAllCache<UserUpdate>;
	notificationCache: ListAllCache<Notification>;

	permissions: string[];

	private endpointPrefix: string;

	constructor(
		api: LeashAPI,
		user: LeashUser,
		endpointPrefix: string,
		options: LeashUserOptions = {}
	) {
		this.api = api;
		this.id = user.ID;
		this.createdAt = new Date(user.CreatedAt);
		this.updatedAt = new Date(user.UpdatedAt);
		if (user.DeletedAt) {
			this.deletedAt = new Date(user.DeletedAt);
		}

		this.email = user.Email;
		this.pendingEmail = user.PendingEmail;
		this.cardId = user.CardID;
		this.name = user.Name;
		this.pronouns = user.Pronouns;
		this.role = user.Role;
		this.type = user.Type;

		// Student-like fields
		this.graduationYear = user.GraduationYear;
		this.major = user.Major;

		// Employee-like fields
		this.department = user.Department;
		this.jobTitle = user.JobTitle;

		this.permissions = user.Permissions;

		this.endpointPrefix = endpointPrefix;

		this.trainingsCache = new ListAllCache((includeDeleted: boolean, noCache: boolean) =>
			this.api.listAll(
				(options: LeashListOptions, noCache: boolean) => this.getTrainings(options, noCache),
				includeDeleted,
				100,
				noCache
			)
		);
		if (user.Trainings) {
			this.trainingsCache.setValue(
				user.Trainings.map(
					(training) =>
						new Training(api, training, `${this.endpointPrefix}/trainings/${training.Name}`)
				)
			);
		} else if (options.withTrainings) {
			this.trainingsCache.setValue([]);
		}

		this.holdsCache = new ListAllCache((includeDeleted: boolean, noCache: boolean) =>
			this.api.listAll(
				(options: LeashListOptions, noCache: boolean) => this.getHolds(options, noCache),
				includeDeleted,
				100,
				noCache
			)
		);
		if (user.Holds) {
			this.holdsCache.setValue(
				user.Holds.map(
					(hold) => new Hold(this.api, hold, `${this.endpointPrefix}/holds/${hold.Name}`)
				)
			);
		} else if (options.withHolds) {
			this.holdsCache.setValue([]);
		}

		this.APIKeysCache = new ListAllCache((includeDeleted: boolean, noCache: boolean) =>
			this.api.listAll(
				(options: LeashListOptions, noCache: boolean) => this.getAPIKeys(options, noCache),
				includeDeleted,
				100,
				noCache
			)
		);
		if (user.APIKeys) {
			this.APIKeysCache.setValue(
				user.APIKeys.map(
					(key) => new APIKey(this.api, key, `${this.endpointPrefix}/api_keys/${key.Key}`)
				)
			);
		} else if (options.withApiKeys) {
			this.APIKeysCache.setValue([]);
		}

		this.userUpdatesCache = new ListAllCache((includeDeleted: boolean, noCache: boolean) =>
			this.api.listAll(
				(options: LeashListOptions, noCache: boolean) => this.getUserUpdates(options, noCache),
				includeDeleted,
				100,
				noCache
			)
		);
		if (user.UserUpdates) {
			this.userUpdatesCache.setValue(
				user.UserUpdates.map((update) => new UserUpdate(this.api, update))
			);
		} else if (options.withUpdates) {
			this.userUpdatesCache.setValue([]);
		}

		this.notificationCache = new ListAllCache((includeDeleted: boolean, noCache: boolean) =>
			this.api.listAll(
				(options: LeashListOptions, noCache: boolean) => this.getNotifications(options, noCache),
				includeDeleted,
				100,
				noCache
			)
		);
		if (user.Notifications) {
			this.notificationCache.setValue(
				user.Notifications.map(
					(notification) =>
						new Notification(
							this.api,
							notification,
							`${this.endpointPrefix}/notifications/${notification.ID}`
						)
				)
			);
		} else {
			this.notificationCache.setValue([]);
		}
	}

	get iconURL(): string {
		if (!this.api.identiconURLCache.has(this.id)) {
			const svg = minidenticon(this.id.toString());
			const blob = new Blob([svg], { type: 'image/svg+xml' });
			this.api.identiconURLCache.set(this.id, URL.createObjectURL(blob));
		}

		return this.api.identiconURLCache.get(this.id) || '';
	}

	get roleNumber(): number {
		switch (this.role) {
			case 'service':
				return Role.USER_ROLE_SERVICE;
			case 'member':
				return Role.USER_ROLE_MEMBER;
			case 'volunteer':
				return Role.USER_ROLE_VOLUNTEER;
			case 'staff':
				return Role.USER_ROLE_STAFF;
			case 'admin':
				return Role.USER_ROLE_ADMIN;
			default:
				throw new Error(`Unknown role ${this.role}`);
		}
	}

	get isStaff(): boolean {
		return this.roleNumber >= Role.USER_ROLE_VOLUNTEER;
	}

	async getTrainings(
		options: LeashListOptions = {},
		noCache = false
	): Promise<LeashListResponse<Training>> {
		const prefix = `${this.endpointPrefix}/trainings`;
		const res = await this.api.leashList<LeashTraining, LeashListOptions>(prefix, options, noCache);
		return {
			total: res.total,
			data: res.data.map(
				(training) => new Training(this.api, training, `${prefix}/${training.Name}`)
			)
		};
	}

	async getAllTrainings(includeDeleted = false, noCache = false): Promise<Training[]> {
		return this.trainingsCache.get(includeDeleted, noCache);
	}

	async getHolds(
		options: LeashListOptions = {},
		noCache = false
	): Promise<LeashListResponse<Hold>> {
		const prefix = `${this.endpointPrefix}/holds`;
		const res = await this.api.leashList<LeashHold, LeashListOptions>(prefix, options, noCache);
		return {
			total: res.total,
			data: res.data.map((hold) => new Hold(this.api, hold, `${prefix}/${hold.Name}`))
		};
	}

	async getAllHolds(includeDeleted = false, noCache = false): Promise<Hold[]> {
		return this.holdsCache.get(includeDeleted, noCache);
	}

	async getAPIKeys(
		options: LeashListOptions = {},
		noCache = false
	): Promise<LeashListResponse<APIKey>> {
		const prefix = `${this.endpointPrefix}/apikeys`;
		const res = await this.api.leashList<LeashAPIKey, LeashListOptions>(prefix, options, noCache);
		return {
			total: res.total,
			data: res.data.map((key) => new APIKey(this.api, key, `${prefix}/${key.Key}`))
		};
	}

	async getAllAPIKeys(includeDeleted = false, noCache = false): Promise<APIKey[]> {
		return this.APIKeysCache.get(includeDeleted, noCache);
	}

	async getUserUpdates(
		options: LeashListOptions = {},
		noCache = false
	): Promise<LeashListResponse<UserUpdate>> {
		const res = await this.api.leashList<LeashUserUpdate, LeashListOptions>(
			`${this.endpointPrefix}/updates`,
			options,
			noCache
		);
		return {
			total: res.total,
			data: res.data.map((update) => new UserUpdate(this.api, update))
		};
	}

	async getAllUserUpdates(includeDeleted = false, noCache = false): Promise<UserUpdate[]> {
		return this.userUpdatesCache.get(includeDeleted, noCache);
	}

	async getNotifications(
		options: LeashListOptions = {},
		noCache = false
	): Promise<LeashListResponse<Notification>> {
		const prefix = `${this.endpointPrefix}/notifications`;
		const res = await this.api.leashList<LeashNotification, LeashListOptions>(
			prefix,
			options,
			noCache
		);
		return {
			total: res.total,
			data: res.data.map(
				(notification) => new Notification(this.api, notification, `${prefix}/${notification.ID}`)
			)
		};
	}

	async getAllNotifications(includeDeleted = false, noCache = false): Promise<Notification[]> {
		return this.notificationCache.get(includeDeleted, noCache);
	}

	async get(options: LeashUserOptions = {}): Promise<User> {
		return new User(
			this.api,
			await this.api.leashGet<LeashUser>(`${this.endpointPrefix}`, options, true),
			this.endpointPrefix,
			options
		);
	}

	async delete(): Promise<void> {
		await this.api.leashFetch(`${this.endpointPrefix}`, 'DELETE', undefined, true);
	}

	async update({
		name,
		pronouns,
		email,
		cardID,
		role,
		type,
		graduationYear,
		major,
		department,
		jobTitle
	}: UserUpdateOptions): Promise<User> {
		if (this.role === 'service') {
			throw new Error('Service users cannot be updated with this method.');
		}

		const updated = await this.api.leashFetch<LeashUser>(`${this.endpointPrefix}`, 'PATCH', {
			name,
			pronouns,
			email,
			card_id: cardID,
			role,
			type,
			graduation_year: graduationYear,
			major,
			department,
			job_title: jobTitle
		});

		return new User(this.api, updated, this.endpointPrefix);
	}

	async updateService({ name, permissions }: ServiceUserUpdateOptions): Promise<User> {
		if (this.role !== 'service') {
			throw new Error('Only service users can be updated with this method.');
		}

		const updated = await this.api.leashFetch<LeashUser>(
			`${this.endpointPrefix}/service`,
			'PATCH',
			{
				name,
				permissions
			}
		);

		return new User(this.api, updated, this.endpointPrefix);
	}

	async createAPIKey({
		description,
		fullAccess,
		permissions
	}: APIKeyCreateOptions): Promise<APIKey> {
		const key = await this.api.leashFetch<LeashAPIKey>(`${this.endpointPrefix}/apikeys`, 'POST', {
			description,
			full_access: fullAccess,
			permissions
		});

		this.APIKeysCache.invalidate();

		return new APIKey(this.api, key, `${this.endpointPrefix}/apikeys/${key.Key}`);
	}

	async getAPIKey(key: string, noCache = false): Promise<APIKey> {
		return new APIKey(
			this.api,
			await this.api.leashGet<LeashAPIKey>(`${this.endpointPrefix}/apikeys/${key}`, {}, noCache),
			`${this.endpointPrefix}/apikeys/${key}`
		);
	}

	async createTraining({ name, level }: TrainingCreateOptions): Promise<Training> {
		const training = await this.api.leashFetch<LeashTraining>(
			`${this.endpointPrefix}/trainings`,
			'POST',
			{
				name,
				level
			}
		);

		this.trainingsCache.invalidate();

		return new Training(this.api, training, `${this.endpointPrefix}/trainings/${training.Name}`);
	}

	async getTraining(name: string, noCache = false): Promise<Training> {
		return new Training(
			this.api,
			await this.api.leashGet<LeashTraining>(
				`${this.endpointPrefix}/trainings/${name}`,
				{},
				noCache
			),
			`${this.endpointPrefix}/trainings/${name}`
		);
	}

	async createHold({
		name,
		reason,
		start,
		end,
		priority,
		resolutionLink
	}: HoldCreateOptions): Promise<Hold> {
		const hold = await this.api.leashFetch<LeashHold>(`${this.endpointPrefix}/holds`, 'POST', {
			name,
			reason,
			start,
			end,
			priority,
			resolution_link: resolutionLink
		});

		this.holdsCache.invalidate();

		return new Hold(this.api, hold, `${this.endpointPrefix}/holds/${hold.Name}`);
	}

	async getHold(name: string, noCache = false): Promise<Hold> {
		return new Hold(
			this.api,
			await this.api.leashGet<LeashHold>(`${this.endpointPrefix}/holds/${name}`, {}, noCache),
			`${this.endpointPrefix}/holds/${name}`
		);
	}

	async createNotification({
		title,
		message,
		link,
		group
	}: NotificationCreateOptions): Promise<Notification> {
		const notification = await this.api.leashFetch<LeashNotification>(
			`${this.endpointPrefix}/notifications`,
			'POST',
			{
				title,
				message,
				link,
				group
			}
		);

		this.notificationCache.invalidate();

		return new Notification(
			this.api,
			notification,
			`${this.endpointPrefix}/notifications/${notification.ID}`
		);
	}

	async getNotification(notificationID: number, noCache = false): Promise<Notification> {
		return new Notification(
			this.api,
			await this.api.leashGet<LeashNotification>(
				`${this.endpointPrefix}/notifications/${notificationID}`,
				{},
				noCache
			),
			`${this.endpointPrefix}/notifications/${notificationID}`
		);
	}

	async getAllPermissions(noCache = false): Promise<string[]> {
		return await this.api.leashGet<string[]>(`${this.endpointPrefix}/permissions`, {}, noCache);
	}

	async checkin(): Promise<CheckinResponse> {
		const res = await this.api.leashGet<LeashCheckinResponse>(
			`${this.endpointPrefix}/checkin`,
			undefined,
			true
		);
		return {
			token: res.token,
			expiresAt: new Date(res.expires_at)
		};
	}
}

export class APIKey {
	private api: LeashAPI;
	key: string;
	createdAt: Date;
	updatedAt: Date;
	deletedAt?: Date;

	private userID: number;
	description: string;
	fullAccess: boolean;
	permissions: string[];

	private endpointPrefix: string;

	constructor(api: LeashAPI, key: LeashAPIKey, endpointPrefix: string) {
		this.api = api;
		this.key = key.Key;
		this.createdAt = new Date(key.CreatedAt);
		this.updatedAt = new Date(key.UpdatedAt);
		if (key.DeletedAt) {
			this.deletedAt = new Date(key.DeletedAt);
		}

		this.userID = key.UserID;
		this.description = key.Description;
		this.fullAccess = key.FullAccess;
		this.permissions = key.Permissions;

		this.endpointPrefix = endpointPrefix;
	}

	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.userID, options, noCache);
	}

	async get(): Promise<APIKey> {
		return new APIKey(
			this.api,
			await this.api.leashGet<LeashAPIKey>(`${this.endpointPrefix}`, {}, true),
			this.endpointPrefix
		);
	}

	async delete(): Promise<void> {
		await this.api.leashFetch(`${this.endpointPrefix}`, 'DELETE', undefined, true);
	}

	async update({ description, fullAccess, permissions }: APIKeyUpdateOptions): Promise<APIKey> {
		const updated = await this.api.leashFetch<LeashAPIKey>(`${this.endpointPrefix}`, 'PATCH', {
			description,
			full_access: fullAccess,
			permissions
		});

		return new APIKey(this.api, updated, this.endpointPrefix);
	}
}

export class Training {
	private api: LeashAPI;
	id: number;
	createdAt: Date;
	updatedAt: Date;
	deletedAt?: Date;

	name: string;
	level: TrainingLevel;

	private userID: number;
	private addedById: number;
	private removedById?: number;

	private endpointPrefix: string;

	constructor(api: LeashAPI, training: LeashTraining, endpointPrefix: string) {
		this.api = api;
		this.id = training.ID;
		this.createdAt = new Date(training.CreatedAt);
		this.updatedAt = new Date(training.UpdatedAt);
		if (training.DeletedAt) {
			this.deletedAt = new Date(training.DeletedAt);
		}

		this.name = training.Name;
		this.level = training.Level;
		this.userID = training.UserID;
		this.addedById = training.AddedBy;
		if (training.RemovedBy) {
			this.removedById = training.RemovedBy;
		}

		this.endpointPrefix = endpointPrefix;
	}

	levelString(): string {
		return trainingLevelToString(this.level);
	}

	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.userID, options, noCache);
	}

	async getAddedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.addedById, options, noCache);
	}

	async getRemovedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		if (!this.removedById) {
			throw new Error('Training has not been removed.');
		}

		return this.api.userFromID(this.removedById, options, noCache);
	}

	async get(): Promise<Training> {
		return new Training(
			this.api,
			await this.api.leashGet<LeashTraining>(`${this.endpointPrefix}`, {}, true),
			this.endpointPrefix
		);
	}

	async delete(): Promise<void> {
		await this.api.leashFetch(`${this.endpointPrefix}`, 'DELETE', undefined, true);
	}
}

export class Hold {
	private api: LeashAPI;
	id: number;
	createdAt: Date;
	updatedAt: Date;
	deletedAt?: Date;

	name: string;
	reason: string;
	start?: Date;
	end?: Date;

	resolutionLink?: string;

	priority: number;

	private userID: number;
	private addedById: number;
	private removedById?: number;

	private endpointPrefix: string;

	constructor(api: LeashAPI, hold: LeashHold, endpointPrefix: string) {
		this.api = api;
		this.id = hold.ID;
		this.createdAt = new Date(hold.CreatedAt);
		this.updatedAt = new Date(hold.UpdatedAt);
		if (hold.DeletedAt) {
			this.deletedAt = new Date(hold.DeletedAt);
		}

		this.name = hold.Name;
		this.reason = hold.Reason;
		if (hold.Start) {
			this.start = new Date(hold.Start);
		}
		if (hold.End) {
			this.end = new Date(hold.End);
		}

		this.userID = hold.UserID;
		this.addedById = hold.AddedBy;
		if (hold.RemovedBy) {
			this.removedById = hold.RemovedBy;
		}

		this.resolutionLink = hold.ResolutionLink;

		this.priority = hold.Priority;

		this.endpointPrefix = endpointPrefix;

		if (this.end && !this.deletedAt && !this.removedById && isAfter(new Date(), this.end)) {
			this.deletedAt = this.end;
			this.removedById = this.addedById;
		}
	}

	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.userID, options, noCache);
	}

	async getAddedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.addedById, options, noCache);
	}

	async getRemovedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		if (!this.removedById) {
			throw new Error('Hold has not been removed.');
		}

		return this.api.userFromID(this.removedById, options, noCache);
	}

	isPending(): boolean {
		const ended = this.end ? isAfter(new Date(), this.end) : false;
		const started = this.start ? isAfter(new Date(), this.start) : true;
		return !started && !ended && this.deletedAt === undefined;
	}

	isActive(): boolean {
		const ended = this.end ? isAfter(new Date(), this.end) : false;
		const started = this.start ? isAfter(new Date(), this.start) : true;
		return started && !ended && this.deletedAt === undefined;
	}

	async get(): Promise<Hold> {
		return new Hold(
			this.api,
			await this.api.leashGet<LeashHold>(`${this.endpointPrefix}`, {}, true),
			this.endpointPrefix
		);
	}

	async delete(): Promise<void> {
		await this.api.leashFetch(`${this.endpointPrefix}`, 'DELETE', undefined, true);
	}

	activeLevel(): number {
		if (this.isActive()) {
			return 0;
		} else if (this.isPending()) {
			return 1;
		} else {
			return 2;
		}
	}
}

export class UserUpdate {
	private api: LeashAPI;
	id: number;
	createdAt: Date;
	updatedAt: Date;
	deletedAt?: Date;

	private userID: number;
	private editedById: number;

	field: string;
	oldValue: string;
	newValue: string;

	constructor(api: LeashAPI, update: LeashUserUpdate) {
		this.api = api;
		this.id = update.ID;
		this.createdAt = new Date(update.CreatedAt);
		this.updatedAt = new Date(update.UpdatedAt);
		if (update.DeletedAt) {
			this.deletedAt = new Date(update.DeletedAt);
		}

		this.userID = update.UserID;
		this.editedById = update.EditedBy;

		this.field = update.Field;
		this.oldValue = update.OldValue;
		this.newValue = update.NewValue;
	}

	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.userID, options, noCache);
	}

	async getEditedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.editedById, options, noCache);
	}
}

export class Notification {
	private api: LeashAPI;
	id: number;
	createdAt: Date;
	updatedAt: Date;
	deletedAt?: Date;

	private userID: number;

	title: string;
	message: string;
	link: string;
	group: string;

	private addedById: number;

	private endpointPrefix: string;

	constructor(api: LeashAPI, notification: LeashNotification, endpointPrefix: string) {
		this.api = api;
		this.id = notification.ID;
		this.createdAt = new Date(notification.CreatedAt);
		this.updatedAt = new Date(notification.UpdatedAt);
		if (notification.DeletedAt) {
			this.deletedAt = new Date(notification.DeletedAt);
		}

		this.userID = notification.UserID;

		this.title = notification.Title;
		this.message = notification.Message;
		this.link = notification.Link;
		this.group = notification.Group;

		this.addedById = notification.AddedBy;

		this.endpointPrefix = endpointPrefix;
	}

	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.userID, options, noCache);
	}

	async getAddedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.addedById, options, noCache);
	}

	async get(): Promise<Notification> {
		return new Notification(
			this.api,
			await this.api.leashGet<LeashNotification>(`${this.endpointPrefix}`, {}, true),
			this.endpointPrefix
		);
	}

	async delete(): Promise<void> {
		await this.api.leashFetch(`${this.endpointPrefix}`, 'DELETE', undefined, true);
	}
}


export class Feed {
	private api: LeashAPI;
	id: number;
	createdAt: Date;
	updatedAt: Date;
	deletedAt?: Date;

	private userID: number;

	title: string;
	message: string;
	link: string;
	group: string;

	private addedById: number;

	private endpointPrefix: string;

	constructor(api: LeashAPI, notification: LeashNotification, endpointPrefix: string) {
		this.api = api;
		this.id = notification.ID;
		this.createdAt = new Date(notification.CreatedAt);
		this.updatedAt = new Date(notification.UpdatedAt);
		if (notification.DeletedAt) {
			this.deletedAt = new Date(notification.DeletedAt);
		}

		this.userID = notification.UserID;

		this.title = notification.Title;
		this.message = notification.Message;
		this.link = notification.Link;
		this.group = notification.Group;

		this.addedById = notification.AddedBy;

		this.endpointPrefix = endpointPrefix;
	}

	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.userID, options, noCache);
	}

	async getAddedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.addedById, options, noCache);
	}

	async get(): Promise<Notification> {
		return new Notification(
			this.api,
			await this.api.leashGet<LeashNotification>(`${this.endpointPrefix}`, {}, true),
			this.endpointPrefix
		);
	}

	async delete(): Promise<void> {
		await this.api.leashFetch(`${this.endpointPrefix}`, 'DELETE', undefined, true);
	}
}


export class FeedMessage {
	private api: LeashAPI;
	id: number;
	createdAt: Date;
	updatedAt: Date;
	deletedAt?: Date;

	private userID: number;

	title: string;
	message: string;
	link: string;
	group: string;

	private addedById: number;

	private endpointPrefix: string;

	constructor(api: LeashAPI, notification: LeashNotification, endpointPrefix: string) {
		this.api = api;
		this.id = notification.ID;
		this.createdAt = new Date(notification.CreatedAt);
		this.updatedAt = new Date(notification.UpdatedAt);
		if (notification.DeletedAt) {
			this.deletedAt = new Date(notification.DeletedAt);
		}

		this.userID = notification.UserID;

		this.title = notification.Title;
		this.message = notification.Message;
		this.link = notification.Link;
		this.group = notification.Group;

		this.addedById = notification.AddedBy;

		this.endpointPrefix = endpointPrefix;
	}

	async getUser(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.userID, options, noCache);
	}

	async getAddedBy(options: LeashUserOptions = {}, noCache = false): Promise<User> {
		return this.api.userFromID(this.addedById, options, noCache);
	}

	async get(): Promise<Notification> {
		return new Notification(
			this.api,
			await this.api.leashGet<LeashNotification>(`${this.endpointPrefix}`, {}, true),
			this.endpointPrefix
		);
	}

	async delete(): Promise<void> {
		await this.api.leashFetch(`${this.endpointPrefix}`, 'DELETE', undefined, true);
	}
}
