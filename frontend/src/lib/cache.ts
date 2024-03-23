import { isAfter, addMilliseconds } from 'date-fns';
import { Mutex } from 'async-mutex';

export class Cached<T> {
	private mutex: Mutex
	private getter: () => Promise<T>;
	private value: Promise<T> | null = null;
	private expiresAt: Date | null = null;

	private defaultTTL: number = 1000 * 30; // 30 seconds

	constructor(
		getter: () => Promise<T>,
		defaultTTL?: number
	) {
		this.mutex = new Mutex();

		this.getter = getter;

		if (defaultTTL) {
			this.defaultTTL = defaultTTL;
		}
	}

	async get(forceRefresh = false): Promise<T> {
		if (forceRefresh) {
			await this.invalidate();
		}

		if (this.expiresAt && isAfter(new Date(), this.expiresAt)) {
			await this.invalidate();
		}


		const release = await this.mutex.acquire();
		if (!this.value) {
			this.value = this.getter().then((value) => {
                this.expiresAt = addMilliseconds(new Date(), this.defaultTTL);
                return value;
            });
		}
		
		release();
		return this.value;
	}

	async invalidate(): Promise<void> {
		this.value = null;
		this.expiresAt = null;
	}

	async setValue(value: T): Promise<void> {
		this.expiresAt = addMilliseconds(new Date(), this.defaultTTL);

		this.value = Promise.resolve(value);
	}
}