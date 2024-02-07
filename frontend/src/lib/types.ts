import { isAfter, addMilliseconds } from 'date-fns';

export class Cached<T> {
	private value: T | null = null;
	private promise: Promise<T> | null = null;
	private expiresAt: Date | null = null;
	private defaultTTL: number = 1000 * 30; // 30 seconds

	constructor(
		private getter: () => Promise<T>,
		defaultTTL?: number
	) {
		if (defaultTTL) {
			this.defaultTTL = defaultTTL;
		}
	}

	async get(expires = true): Promise<T> {
		if (this.value) {
			if (!this.expiresAt || isAfter(new Date(), this.expiresAt)) {
				return this.value;
			}
		}

		if (this.promise) {
			return this.promise;
		}

		this.promise = this.getter().then((value) => {
			this.value = value;
			if (expires) {
				this.expiresAt = addMilliseconds(new Date(), this.defaultTTL);
			} else {
				this.expiresAt = null;
			}
			return value;
		});

		return this.promise;
	}

	async invalidate(): Promise<void> {
		this.value = null;
		this.promise = null;
		this.expiresAt = null;
	}

	async setValue(value: T, expires = true): Promise<void> {
		this.value = value;
		if (expires) {
			this.expiresAt = addMilliseconds(new Date(), this.defaultTTL);
		} else {
			this.expiresAt = null;
		}

		this.promise = Promise.resolve(value);
	}
}

export type ConvertFields<T, V> = T extends string | number | boolean | undefined
	? V
	: T extends object
		? { [K in keyof T]: ConvertFields<T[K], V> }
		: never;
