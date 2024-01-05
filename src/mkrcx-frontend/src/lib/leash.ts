import Cookies from 'js-cookie';
import { Cached } from './types';
import { isAfter } from "date-fns";
import { minidenticon } from 'minidenticons'
import { PUBLIC_LEASH_ENDPOINT as LEASH_ENDPOINT } from '$env/static/public';

const requestCache: Map<string, Cached<unknown>> = new Map();

async function leashFetch<T extends object>(
	endpoint: string,
	method: string,
	body?: object,
	noReponse = false
): Promise<T> {
	const r = await fetch(`${LEASH_ENDPOINT}${endpoint}`, {
		method: method,
		headers: {
			Authorization: `Bearer ${Cookies.get('token')}`,
			'Content-Type': 'application/json'
		},
		redirect: 'follow',
		mode: 'cors',
		cache: 'no-cache',
		credentials: 'same-origin',
		body: JSON.stringify(body)
	});
	return await (r.status !== 200
		? Promise.reject(new Error(await r.text()))
		: noReponse
		? r.text()
		: r.json());
}

export enum Role {
	USER_ROLE_SERVICE = 0,
	USER_ROLE_MEMBER = 1,
	USER_ROLE_VOLUNTEER = 2,
	USER_ROLE_STAFF = 3,
	USER_ROLE_ADMIN = 4,
}

interface LeashUser {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	DeletedAt?: string;

	Email: string;
	PendingEmail?: string;
	CardID: number;
	Name: string;
	Role: string;
	Type: string;
	GraduationYear: number;
	Major: string;

	Trainings?: LeashTraining[];
	Holds?: LeashHold[];
	APIKeys?: LeashAPIKey[];
	UserUpdates?: LeashUserUpdate[];

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

interface LeashTraining {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	DeletedAt?: string;

	UserID: number;
	TrainingType: string;
	AddedBy: number;
	RemovedBy?: number;
}

interface LeashHold {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	DeletedAt?: string;

	UserID: number;
	HoldType: string;
	Reason: string;
	HoldStart?: string;
	HoldEnd?: string;
	AddedBy: number;
	RemovedBy?: number;
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

export interface UserCreateOptions {
    email: string;
    name: string;
    role: string;
    type: string;
    graduationYear: number;
    major: string;
}

export interface UserUpdateOptions {
    name?: string;
    email?: string;
    cardID?: number;
    enabled?: boolean;
    role?: string;
    type?: string;
    graduationYear?: number;
    major?: string;
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
    trainingType: string;
}

export interface HoldCreateOptions {
    holdType: string;
    reason: string;
    holdStart?: string;
    holdEnd?: string;
}

export interface LeashListResponse<T> {
	total: number;
	data: T[];
}

export interface LeashUserOptions {
    withTrainings?: boolean;
    withHolds?: boolean;
    withApiKeys?: boolean;
    withUpdates?: boolean;
}

export interface LeashListOptions {
	offset?: number;
	limit?: number;
	includeDeleted?: boolean;
};

export interface LeashUserSearchOptions extends LeashListOptions, LeashUserOptions {
    showService?: boolean;
}


type LeashListGetter<T> = (options: LeashListOptions) => Promise<LeashListResponse<T>>;

const camelToSnakeCase: (str: string) => string = str => str.replace(/[A-Z]/g, letter => `_${letter.toLowerCase()}`);

async function leashGet<T extends object>(
    endpoint: string,
    options: object = {},
    noCache = false,
): Promise<T> {
    let args = Object.entries(options).map(([key, value]) => `${camelToSnakeCase(key)}=${value}`).join('&');
    if (args.length > 0) {
        args = `?${args}`;
    }

    const link = `${endpoint}${args}`;

    if (!requestCache.has(link)) {
        requestCache.set(link, new Cached(() => leashFetch<T>(link, 'GET')));
    }

    if (noCache) {
        requestCache.get(link)?.invalidate();
    }

    return (requestCache.get(link)?.get() || {}) as T;
}

async function leashList<T, O extends LeashListOptions>(
    endpoint: string,
    options: O | Record<string, never> = {},
    noCache = false,
): Promise<LeashListResponse<T>> {
    return leashGet<LeashListResponse<T>>(endpoint, options, noCache);
}

async function listAll<T>(getter: LeashListGetter<T>, includeDeleted = false, limit = 100): Promise<T[]> {
	let offset = 0;
	let result: T[] = [];
	let currentResult: LeashListResponse<T>;
	do {
		currentResult = await getter({
			offset,
			limit,
			includeDeleted
		});
		result = result.concat(currentResult.data);
		offset += limit;
	} while (currentResult.total > offset);

	return result;
}

const identiconURLCache = new Map<number, string>();

export class User {
	id: number;
	createdAt: Date;
	updatedAt: Date;
	deletedAt?: Date;

	email: string;
	pendingEmail?: string;
	cardId: number;
	name: string;
	role: string;
	type: string;
	graduationYear: number;
	major: string;

	private trainingsCache: Cached<Training[]>;
	private holdsCache: Cached<Hold[]>;
	private APIKeysCache: Cached<APIKey[]>;
	private userUpdatesCache: Cached<UserUpdate[]>;

	permissions: string[];

    private endpointPrefix: string;

	constructor(user: LeashUser, endpointPrefix: string) {
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
		this.role = user.Role;
		this.type = user.Type;
		this.graduationYear = user.GraduationYear;
		this.major = user.Major;
		
		this.trainingsCache = new Cached(() => listAll((options) => this.getTrainings(options)));
		if (user.Trainings) {
			this.trainingsCache.setValue(user.Trainings.map((training) => new Training(training, `${this.endpointPrefix}/trainings/${training.TrainingType}`)));
		}

		this.holdsCache = new Cached(() => listAll((options) => this.getHolds(options)));
		if (user.Holds) {
			this.holdsCache.setValue(user.Holds.map((hold) => new Hold(hold, `${this.endpointPrefix}/holds/${hold.HoldType}`)));
		}

		this.APIKeysCache = new Cached(() => listAll((options) => this.getAPIKeys(options)));
		if (user.APIKeys) {
			this.APIKeysCache.setValue(user.APIKeys.map((key) => new APIKey(key, `${this.endpointPrefix}/api_keys/${key.Key}`)));
		}

		this.userUpdatesCache = new Cached(() => listAll((options) => this.getUserUpdates(options)));
		if (user.UserUpdates) {
			this.userUpdatesCache.setValue(user.UserUpdates.map((update) => new UserUpdate(update)));
		}

		this.permissions = user.Permissions;
            
        this.endpointPrefix = endpointPrefix;
	}

    get iconURL(): string {
        if (!identiconURLCache.has(this.id)) {
            const svg = minidenticon(this.id.toString());
            identiconURLCache.set(this.id, 'data:image/svg+xml;utf8,' + encodeURIComponent(svg))
        }

        return identiconURLCache.get(this.id) || '';
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

	async getTrainings(options: LeashListOptions = {}): Promise<LeashListResponse<Training>> {
        const prefix = `${this.endpointPrefix}/trainings`;
        const res = await leashList<LeashTraining, LeashListOptions>(prefix, options);
        return {
            total: res.total,
            data: res.data.map((training) => new Training(training, `${prefix}/${training.TrainingType}`))
        };
    }

    async getAllTrainings(): Promise<Training[]> {
        return this.trainingsCache.get();
    }

    async getHolds(options: LeashListOptions = {}): Promise<LeashListResponse<Hold>> {
        const prefix = `${this.endpointPrefix}/holds`;
        const res = await leashList<LeashHold, LeashListOptions>(prefix, options);
        return {
            total: res.total,
            data: res.data.map((hold) => new Hold(hold, `${prefix}/${hold.HoldType}`))
        };
    }

    async getAllHolds(): Promise<Hold[]> {
        return this.holdsCache.get();
    }

    async getAPIKeys(options: LeashListOptions = {}): Promise<LeashListResponse<APIKey>> {
        const prefix = `${this.endpointPrefix}/apikeys`;
        const res = await leashList<LeashAPIKey, LeashListOptions>(prefix, options);
        return {
            total: res.total,
            data: res.data.map((key) => new APIKey(key, `${prefix}/${key.Key}`))
        };
    }

    async getAllAPIKeys(): Promise<APIKey[]> {
        return this.APIKeysCache.get();
    }

    async getUserUpdates(options: LeashListOptions = {}): Promise<LeashListResponse<UserUpdate>> {
        const res = await leashList<LeashUserUpdate, LeashListOptions>(`${this.endpointPrefix}/updates`, options);
        return {
            total: res.total,
            data: res.data.map((update) => new UserUpdate(update))
        };
    }

    async getAllUserUpdates(): Promise<UserUpdate[]> {
        return this.userUpdatesCache.get();
    }

    async get(): Promise<User> {
        return new User(await leashGet<LeashUser>(`${this.endpointPrefix}`, {}, true), this.endpointPrefix);
    }

    async delete(): Promise<void> {
        leashFetch(`${this.endpointPrefix}`, 'DELETE');
    }

    async update({ name, email, cardID, enabled, role, type, graduationYear, major }: UserUpdateOptions): Promise<User> {
        if (this.role === 'service') {
            throw new Error('Service users cannot be updated with this method.');
        }

        const updated = await leashFetch<LeashUser>(`${this.endpointPrefix}`, 'PATCH', {
            name,
            email,
            card_id: cardID,
            enabled,
            role,
            type,
            graduation_year: graduationYear,
            major
        });

        requestCache.get(this.endpointPrefix)?.setValue(updated);

        return new User(updated, this.endpointPrefix);
    }

    async updateService({ name, permissions }: ServiceUserUpdateOptions): Promise<User> {
        if (this.role !== 'service') {
            throw new Error('Only service users can be updated with this method.');
        }

        const updated = await leashFetch<LeashUser>(`${this.endpointPrefix}`, 'PATCH', {
            name,
            permissions
        });

        requestCache.get(this.endpointPrefix)?.setValue(updated);

        return new User(updated, this.endpointPrefix);
    }

    async createAPIKey({ description, fullAccess, permissions }: APIKeyCreateOptions): Promise<APIKey> {
        const key = await leashFetch<LeashAPIKey>(`${this.endpointPrefix}/apikeys`, 'POST', {
            description,
            full_access: fullAccess,
            permissions
        });

        this.APIKeysCache.invalidate();

        return new APIKey(key, `${this.endpointPrefix}/apikeys/${key.Key}`);
    }

    async getAPIKey(key: string): Promise<APIKey> {
        return new APIKey(await leashGet<LeashAPIKey>(`${this.endpointPrefix}/apikeys/${key}`), `${this.endpointPrefix}/apikeys/${key}`);
    }

    async createTraining({ trainingType }: TrainingCreateOptions): Promise<Training> {
        const training = await leashFetch<LeashTraining>(`${this.endpointPrefix}/trainings`, 'POST', {
            training_type: trainingType
        });

        this.trainingsCache.invalidate();

        return new Training(training, `${this.endpointPrefix}/trainings/${training.TrainingType}`);
    }

    async getTraining(trainingType: string): Promise<Training> {
        return new Training(await leashGet<LeashTraining>(`${this.endpointPrefix}/trainings/${trainingType}`), `${this.endpointPrefix}/trainings/${trainingType}`);
    }

    async createHold({ holdType, reason, holdStart, holdEnd }: HoldCreateOptions): Promise<Hold> {
        const hold = await leashFetch<LeashHold>(`${this.endpointPrefix}/holds`, 'POST', {
            hold_type: holdType,
            reason,
            hold_start: holdStart,
            hold_end: holdEnd
        });

        this.holdsCache.invalidate();

        return new Hold(hold, `${this.endpointPrefix}/holds/${hold.HoldType}`);
    }

    async getHold(holdType: string): Promise<Hold> {
        return new Hold(await leashGet<LeashHold>(`${this.endpointPrefix}/holds/${holdType}`), `${this.endpointPrefix}/holds/${holdType}`);
    }

    static async create({ email, name, role, type, graduationYear, major }: UserCreateOptions): Promise<User> {
        const user = await leashFetch<LeashUser>(`/api/users`, 'POST', {
            email,
            name,
            role,
            type,
            graduation_year: graduationYear,
            major
        });

        return new User(user, `/api/users/${user.ID}`);
    }
    
    static async createService({ name, permissions }: ServiceUserCreateOptions): Promise<User> {
        const user = await leashFetch<LeashUser>(`/api/users`, 'POST', {
            name,
            permissions
        });

        return new User(user, `/api/users/${user.ID}`);
    }

    static async search(query: string, options: LeashUserSearchOptions = {}): Promise<LeashListResponse<User>> {
        interface LeashUserSearchOptionsWhole extends LeashUserSearchOptions {
            query: string;
        }

        const optionsWhole: LeashUserSearchOptionsWhole = {
            ...options,
            query
        };

        const res = await leashList<LeashUser, LeashUserSearchOptionsWhole>(`/api/users/search/`, optionsWhole);
        
        return {
            total: res.total,
            data: res.data.map((user) => new User(user, `/api/users/${user.ID}`))
        };
    }

    static async fromID(id: number): Promise<User> {
        return leashGet<LeashUser>(`/api/users/${id}`).then((user) => new User(user, `/api/users/${id}`));
    }

    static async self(): Promise<User> {
        return leashGet<LeashUser>(`/api/users/self`).then((user) => new User(user, `/api/users/self`));
    }

    static async fromEmail(email: string): Promise<User> {
        return leashGet<LeashUser>(`/api/users/get/email/${email}`).then((user) => new User(user, `/api/users/${user.ID}`));
    }

    static async fromCardID(cardID: number): Promise<User> {
        return leashGet<LeashUser>(`/api/users/get/card/${cardID}`).then((user) => new User(user, `/api/users/${user.ID}`));
    }
}

export class APIKey {
    key: string;
    createdAt: Date;
    updatedAt: Date;
    deletedAt?: Date;

    private userID: number;
    description: string;
    fullAccess: boolean;
    permissions: string[];
    
    private endpointPrefix: string;

    constructor(key: LeashAPIKey, endpointPrefix: string) {
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
    
    async getUser(): Promise<User> {
        return User.fromID(this.userID);
    }

    async get(): Promise<APIKey> {
        return new APIKey(await leashGet<LeashAPIKey>(`${this.endpointPrefix}`, {}, true), this.endpointPrefix);
    }

    async delete(): Promise<void> {
        leashFetch(`${this.endpointPrefix}`, 'DELETE');
    }

    async update({ description, fullAccess, permissions }: APIKeyUpdateOptions): Promise<APIKey> {
        const updated = await leashFetch<LeashAPIKey>(`${this.endpointPrefix}`, 'PATCH', {
            description,
            full_access: fullAccess,
            permissions
        });

        requestCache.get(this.endpointPrefix)?.setValue(updated);

        return new APIKey(updated, this.endpointPrefix);
    }

    static async fromKey(key: string): Promise<APIKey> {
        return leashGet<LeashAPIKey>(`/api/apikeys/${key}`).then((key) => new APIKey(key, `/api/apikeys/${key.Key}`));
    }
}

export class Training {
	id: number;
	createdAt: Date;
	updatedAt: Date;
	deletedAt?: Date;

	trainingType: string;

    private userID: number;
	private addedById: number;
	private removedById?: number;

    private endpointPrefix: string;
    
	constructor(training: LeashTraining, endpointPrefix: string) {
		this.id = training.ID;
		this.createdAt = new Date(training.CreatedAt);
		this.updatedAt = new Date(training.UpdatedAt);
		if (training.DeletedAt) {
			this.deletedAt = new Date(training.DeletedAt);
		}

		this.trainingType = training.TrainingType;
		this.userID = training.UserID;
		this.addedById = training.AddedBy;
		if (training.RemovedBy) {
			this.removedById = training.RemovedBy;
		}

        this.endpointPrefix = endpointPrefix;
	}

    async getUser(): Promise<User> {
        return User.fromID(this.userID);
    }
    
    async getAddedBy(): Promise<User> {
        return User.fromID(this.addedById);
    }

    async getRemovedBy(): Promise<User | undefined> {
        if (!this.removedById) {
            return undefined;
        }

        return User.fromID(this.removedById);
    }

    async get(): Promise<Training> {
        return new Training(await leashGet<LeashTraining>(`${this.endpointPrefix}`, {}, true), this.endpointPrefix);
    }

    async delete(): Promise<void> {
        leashFetch(`${this.endpointPrefix}`, 'DELETE');
    }

    static async fromID(id: number): Promise<Training> {
        return leashGet<LeashTraining>(`/api/trainings/${id}`).then((training) => new Training(training, `/api/trainings/${id}`));
    }
}

export class Hold {
    id: number;
    createdAt: Date;
    updatedAt: Date;
    deletedAt?: Date;

    holdType: string;
    reason: string;
    holdStart?: Date;
    holdEnd?: Date;

    private userID: number;
    private addedById: number;
    private removedById?: number;

    private endpointPrefix: string;
    
    constructor(hold: LeashHold, endpointPrefix: string) {
        this.id = hold.ID;
        this.createdAt = new Date(hold.CreatedAt);
        this.updatedAt = new Date(hold.UpdatedAt);
        if (hold.DeletedAt) {
            this.deletedAt = new Date(hold.DeletedAt);
        }

        this.holdType = hold.HoldType;
        this.reason = hold.Reason;
        if (hold.HoldStart) {
            this.holdStart = new Date(hold.HoldStart);
        }
        if (hold.HoldEnd) {
            this.holdEnd = new Date(hold.HoldEnd);
        }

        this.userID = hold.UserID;
        this.addedById = hold.AddedBy;
        if (hold.RemovedBy) {
            this.removedById = hold.RemovedBy;
        }

        this.endpointPrefix = endpointPrefix;
    }

    async getUser(): Promise<User> {
        return User.fromID(this.userID);
    }
    
    async getAddedBy(): Promise<User> {
        return User.fromID(this.addedById);
    }

    async getRemovedBy(): Promise<User | undefined> {
        if (!this.removedById) {
            return undefined;
        }

        return User.fromID(this.removedById);
    }

    get isActive(): boolean {
        const ended = this.holdEnd ? isAfter(new Date(), this.holdEnd) : false;
        const started = this.holdStart ? isAfter(new Date(), this.holdStart) : true;
        return started && !ended;
    }

    async get(): Promise<Hold> {
        return new Hold(await leashGet<LeashHold>(`${this.endpointPrefix}`, {}, true), this.endpointPrefix);
    }

    async delete(): Promise<void> {
        leashFetch(`${this.endpointPrefix}`, 'DELETE');
    }

    static async fromID(id: number): Promise<Hold> {
        return leashGet<LeashHold>(`/api/holds/${id}`).then((hold) => new Hold(hold, `/api/holds/${id}`));
    }
}

export class UserUpdate {
    id: number;
    createdAt: Date;
    updatedAt: Date;
    deletedAt?: Date;

    private userID: number;
    private editedById: number;

    field: string;
    oldValue: string;
    newValue: string;

    constructor(update: LeashUserUpdate) {
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

    async getUser(): Promise<User> {
        return User.fromID(this.userID);
    }

    async getEditedBy(): Promise<User> {
        return User.fromID(this.editedById);
    }
}

interface LeashTokenRefresh {
	token: string;
	expires_at: string;
}

export async function refreshTokens(): Promise<boolean> {
	try {
		const refresh = await leashFetch<LeashTokenRefresh>(`/auth/refresh`, 'GET');
		Cookies.set('token', refresh.token, {
			expires: new Date(refresh.expires_at),
			sameSite: 'strict'
		});
		return true;
	} catch (e) {
		return false;
	}
}

export async function validateToken(): Promise<boolean> {
	try {
		await leashFetch(`/auth/validate`, 'GET', undefined, true);
		return true;
	} catch (e) {
		return false;
	}
}

export async function login(login: string, return_to: string): Promise<void> {
    if (!return_to) {
        return_to = window.location.href;
    }

    const state = btoa(return_to);

	window.location.href = `${LEASH_ENDPOINT}/auth/login?return=${login}&state=${state}`;
}

export async function logout(return_to: string): Promise<void> {
    const token = Cookies.get('token') || '';

    if (!return_to) {
        return_to = window.location.href;
    }

    Cookies.remove('token');

    window.location.href = `${LEASH_ENDPOINT}/auth/logout?token=${token}&return=${return_to}`;
}