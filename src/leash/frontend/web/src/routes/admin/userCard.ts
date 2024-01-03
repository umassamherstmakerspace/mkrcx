import { CheckCircled, CrossCircled, MinusCircled, Pencil1, PlusCircled } from 'radix-icons-svelte';
import type { Dayjs } from 'dayjs';
import type { Training, User } from '$lib/src/leash';

export interface UserEvent {
	timestamp: Dayjs;
	action: 'created' | 'deleted';
}

export interface TrainingItem {
	timestamp: Dayjs;
	trainingType: string;
	active: boolean;
	action: 'created' | 'deleted';
	addedBy: User;
	removedBy?: User;
}

export interface UserUpdate {
	timestamp: Dayjs;
	field: string;
	oldValue: string;
	newValue: string;
	editedBy: User;
}

export type TimelineElement =
	| {
			elementType: 'user';
			userEvent: UserEvent;
	  }
	| {
			elementType: 'training';
			trainingItem: TrainingItem;
	  }
	| {
			elementType: 'userUpdate';
			userUpdate: UserUpdate;
	  };

export class UserTimelineItem {
	element: TimelineElement;

	get timestamp() {
		switch (this.element.elementType) {
			case 'user':
				return this.element.userEvent.timestamp;
			case 'training':
				return this.element.trainingItem.timestamp;
			case 'userUpdate':
				return this.element.userUpdate.timestamp;
		}
	}

	constructor(element: TimelineElement) {
		this.element = element;
	}

	static fromUserEvent(userEvent: UserEvent): UserTimelineItem {
		return new UserTimelineItem({ elementType: 'user', userEvent });
	}

	static fromTrainingItem(trainingItem: TrainingItem): UserTimelineItem {
		return new UserTimelineItem({ elementType: 'training', trainingItem });
	}

	static fromUserUpdate(userUpdate: UserUpdate): UserTimelineItem {
		return new UserTimelineItem({ elementType: 'userUpdate', userUpdate });
	}

	getBullet() {
		switch (this.element.elementType) {
			case 'user':
				if (this.element.userEvent.action === 'created') {
					return PlusCircled;
				} else {
					return MinusCircled;
				}
			case 'training':
				if (this.element.trainingItem.action === 'created') {
					return CheckCircled;
				} else {
					return CrossCircled;
				}
			case 'userUpdate':
				return Pencil1;
		}
	}

	getBulletColor() {
		switch (this.element.elementType) {
			case 'user':
				if (this.element.userEvent.action === 'created') {
					return 'green';
				} else {
					return 'red';
				}
			case 'training':
				if (this.element.trainingItem.action === 'created') {
					return 'green';
				} else {
					return 'red';
				}
			case 'userUpdate':
				return 'blue';
		}
	}

	getTitle() {
		switch (this.element.elementType) {
			case 'user':
				if (this.element.userEvent.action === 'created') {
					return 'User created';
				} else {
					return 'User deleted';
				}
			case 'training':
				if (this.element.trainingItem.action === 'created') {
					return 'Training added';
				} else {
					return 'Training removed';
				}
			case 'userUpdate':
				return 'User updated';
		}
	}

	getSubtitle() {
		switch (this.element.elementType) {
			case 'user':
				return '';
			case 'training':
				if (this.element.trainingItem.action === 'created') {
					return `Added ${this.element.trainingItem.trainingType} training`;
				} else {
					return `Removed ${this.element.trainingItem.trainingType} training`;
				}
			case 'userUpdate':
				return `Updated ${this.element.userUpdate.field} from ${this.element.userUpdate.oldValue} to ${this.element.userUpdate.newValue}`;
		}
	}
}

export interface UserInfo {
	user: User;
	trainings: Training[];
	timelineItems: UserTimelineItem[];
}

export async function initalizeUserInfo(user: User): Promise<UserInfo> {
	console.log('user', user);
	const trainings = await user.getAllTrainings();
	const updates = await user.getAllUserUpdates();

	const timelineItems: UserTimelineItem[] = [];

	for (const training of trainings) {
		timelineItems.push(
			UserTimelineItem.fromTrainingItem({
				timestamp: training.createdAt,
				trainingType: training.trainingType,
				active: !training.deletedAt,
				action: 'created',
				addedBy: await training.getAddedBy()
			})
		);

		if (training.deletedAt) {
			timelineItems.push(
				UserTimelineItem.fromTrainingItem({
					timestamp: training.deletedAt,
					trainingType: training.trainingType,
					active: true,
					action: 'deleted',
					addedBy: await training.getAddedBy(),
					removedBy: await training.getRemovedBy()
				})
			);
		}
	}

	trainings.sort((a, b) => {
		if (!a.deletedAt && b.deletedAt) {
			return -1;
		} else if (a.deletedAt && !b.deletedAt) {
			return 1;
		} else {
			return a.trainingType.localeCompare(b.trainingType);
		}
	});

	for (const update of updates) {
		timelineItems.push(
			UserTimelineItem.fromUserUpdate({
				timestamp: update.createdAt,
				field: update.field,
				oldValue: update.oldValue,
				newValue: update.newValue,
				editedBy: await update.getEditedBy()
			})
		);
	}

	timelineItems.push(
		UserTimelineItem.fromUserEvent({
			timestamp: user.createdAt,
			action: 'created'
		})
	);

	if (user.deletedAt) {
		timelineItems.push(
			UserTimelineItem.fromUserEvent({
				timestamp: user.deletedAt,
				action: 'deleted'
			})
		);
	}

	timelineItems.sort((a, b) => a.timestamp.diff(b.timestamp));

	return {
		user,
		trainings,
		timelineItems
	};
}
