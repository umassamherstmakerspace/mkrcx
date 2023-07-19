import type { Dayjs } from 'dayjs';
import { createTraining, getTrainings, getUserById, getUserUpdates, removeTraining } from './leash';
import dayjs from 'dayjs';

export interface LeashTraining {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	DeletedAt?: string;

	UserID: number;
	TrainingType: string;
	AddedBy: number;
	RemovedBy?: number;
}

export class Training {
	id: number;
	createdAt: Dayjs;
	updatedAt: Dayjs;
	deletedAt?: Dayjs;

	user: User;
	trainingType: string;

	private addedById: number;
	private removedById?: number;

	constructor(training: LeashTraining, user: User) {
		this.id = training.ID;
		this.createdAt = dayjs(training.CreatedAt);
		this.updatedAt = dayjs(training.UpdatedAt);
		if (training.DeletedAt) {
			this.deletedAt = dayjs(training.DeletedAt);
		}

		this.trainingType = training.TrainingType;
		this.user = user;
		this.addedById = training.AddedBy;
		if (training.RemovedBy) {
			this.removedById = training.RemovedBy;
		}
	}

	async getAddedBy(userCache?: Map<number, User>): Promise<User> {
		if (userCache && userCache.has(this.addedById)) {
			return userCache.get(this.addedById);
		}

		return getUserById(this.addedById).then((user) => {
			if (userCache) {
				userCache.set(this.addedById, user);
			}

			return user;
		});
	}

	async getRemovedBy(userCache?: Map<number, User>): Promise<User> {
		if (!this.removedById) {
			throw new Error('Training has not been removed');
		}

		if (userCache && userCache.has(this.addedById)) {
			return userCache.get(this.removedById);
		}

		return getUserById(this.removedById).then((user) => {
			if (userCache) {
				userCache.set(this.removedById, user);
			}

			return user;
		});
	}

	async remove(): Promise<void> {
		await removeTraining(this.user.email, this.trainingType);
	}
}

export interface LeashUserUpdate {
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

export class UserUpdate {
	id: number;
	createdAt: Dayjs;
	updatedAt: Dayjs;
	deletedAt?: Dayjs;

	user: User;
	field: string;
	oldValue: string;
	newValue: string;

	private editedById: number;

	constructor(training: LeashUserUpdate, user: User) {
		this.id = training.ID;
		this.createdAt = dayjs(training.CreatedAt);
		this.updatedAt = dayjs(training.UpdatedAt);
		if (training.DeletedAt) {
			this.deletedAt = dayjs(training.DeletedAt);
		}

		this.user = user;
		this.field = training.Field;
		this.oldValue = training.OldValue;
		this.newValue = training.NewValue;
		this.editedById = training.EditedBy;
	}

	async getEditedBy(userCache?: Map<number, User>): Promise<User> {
		if (userCache && userCache.has(this.editedById)) {
			return userCache.get(this.editedById);
		}

		return getUserById(this.editedById).then((user) => {
			if (userCache) {
				userCache.set(this.editedById, user);
			}

			return user;
		});
	}
}

export interface LeashUser {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	DeletedAt?: string;

	Email: string;
	Admin: boolean;
	Role: string;
	FirstName: string;
	LastName: string;
	GraduationYear: number;
	Type: string;
	Major: string;
	Enabled: boolean;
	Trainings: LeashTraining[];
	UserUpdates: LeashUserUpdate[];
}

export class User {
	id: number;
	createdAt: Dayjs;
	updatedAt: Dayjs;
	deletedAt?: Dayjs;

	email: string;
	admin: boolean;
	role: string;
	firstName: string;
	lastName: string;
	graduationYear: number;
	type: string;
	major: string;
	enabled: boolean;
	trainings?: Training[];
	userUpdates?: UserUpdate[];

	get name(): string {
		return `${this.firstName} ${this.lastName}`.trim();
	}

	constructor(user: LeashUser) {
		this.id = user.ID;
		this.createdAt = dayjs(user.CreatedAt);
		this.updatedAt = dayjs(user.UpdatedAt);
		if (user.DeletedAt) {
			this.deletedAt = dayjs(user.DeletedAt);
		}

		this.email = user.Email;
		this.admin = user.Admin;
		this.role = user.Role;
		this.firstName = user.FirstName;
		this.lastName = user.LastName;
		this.graduationYear = user.GraduationYear;
		this.type = user.Type;
		this.major = user.Major;
		this.enabled = user.Enabled;
		if (user.Trainings) {
			this.trainings = user.Trainings.map((training) => new Training(training, this));
		}

		if (user.UserUpdates) {
			this.userUpdates = user.UserUpdates.map((update) => new UserUpdate(update, this));
		}
	}

	async getTrainings(): Promise<Training[]> {
		return getTrainings(this.email).then((trainings: LeashTraining[]) => {
			return trainings.map((training) => new Training(training, this));
		});
	}

	async getUserUpdates(): Promise<UserUpdate[]> {
		return getUserUpdates(this.email).then((updates: LeashUserUpdate[]) => {
			return updates.map((update) => new UserUpdate(update, this));
		});
	}

	async createTraining(trainingType: string): Promise<void> {
		return await createTraining(this.email, trainingType);
	}
}
