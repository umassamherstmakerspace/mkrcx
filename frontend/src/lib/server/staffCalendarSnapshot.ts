import { Worker } from 'node:worker_threads';
import { resolve } from 'node:path';
import type { EventInput } from '@fullcalendar/core';

const DEFAULT_REFRESH_INTERVAL_MS = 10 * 60 * 1000;
const WORKER_STARTUP_TIMEOUT_MS = 90 * 1000;
const GENERIC_REFRESH_ERROR = 'Failed to refresh staff calendar snapshot.';
const GENERIC_READ_ERROR = 'Failed to read staff calendar snapshot.';

type WorkerMessage =
	| { type: 'ready' }
	| { type: 'startup-error' }
	| { type: 'result'; id: number; events: EventInput[] }
	| { type: 'request-error'; id: number };

export interface StaffCalendarWorkerClient {
	ready(): Promise<void>;
	getEventsBetween(start: Date, end: Date): Promise<EventInput[]>;
	retire(): void;
}

export type StaffCalendarWorkerFactory = (endpoint: string) => StaffCalendarWorkerClient;

class NodeStaffCalendarWorkerClient implements StaffCalendarWorkerClient {
	private worker: Worker;
	private readyPromise: Promise<void>;
	private resolveReady!: () => void;
	private rejectReady!: (reason: Error) => void;
	private readySettled = false;
	private retired = false;
	private nextRequestID = 1;
	private pending = new Map<
		number,
		{ resolve: (events: EventInput[]) => void; reject: (reason: Error) => void }
	>();
	private startupTimeout: ReturnType<typeof setTimeout>;

	constructor(endpoint: string) {
		this.readyPromise = new Promise((resolveReady, rejectReady) => {
			this.resolveReady = resolveReady;
			this.rejectReady = rejectReady;
		});
		this.worker = new Worker(resolve(process.cwd(), 'build/workers/staff-calendar.mjs'), {
			workerData: { endpoint }
		});
		this.startupTimeout = setTimeout(() => {
			this.fail(new Error(GENERIC_REFRESH_ERROR));
		}, WORKER_STARTUP_TIMEOUT_MS);
		this.startupTimeout.unref();

		this.worker.on('message', (message: WorkerMessage) => this.handleMessage(message));
		this.worker.on('error', () => this.fail(new Error(GENERIC_REFRESH_ERROR)));
		this.worker.on('exit', () => {
			if (!this.retired) this.fail(new Error(GENERIC_REFRESH_ERROR));
		});
	}

	ready(): Promise<void> {
		return this.readyPromise;
	}

	getEventsBetween(start: Date, end: Date): Promise<EventInput[]> {
		if (this.retired) return Promise.reject(new Error(GENERIC_READ_ERROR));

		const id = this.nextRequestID++;
		return new Promise((resolveRequest, rejectRequest) => {
			this.pending.set(id, { resolve: resolveRequest, reject: rejectRequest });
			try {
				this.worker.postMessage({
					type: 'between',
					id,
					start: start.getTime(),
					end: end.getTime()
				});
			} catch {
				this.pending.delete(id);
				rejectRequest(new Error(GENERIC_READ_ERROR));
			}
		});
	}

	retire(): void {
		this.retired = true;
		this.terminateWhenIdle();
	}

	private handleMessage(message: WorkerMessage): void {
		if (message.type === 'ready') {
			this.settleReady();
			return;
		}
		if (message.type === 'startup-error') {
			this.fail(new Error(GENERIC_REFRESH_ERROR));
			return;
		}

		const request = this.pending.get(message.id);
		if (!request) return;
		this.pending.delete(message.id);
		if (message.type === 'result') {
			request.resolve(message.events);
		} else {
			request.reject(new Error(GENERIC_READ_ERROR));
		}
		this.terminateWhenIdle();
	}

	private settleReady(error?: Error): void {
		if (this.readySettled) return;
		this.readySettled = true;
		clearTimeout(this.startupTimeout);
		if (error) this.rejectReady(error);
		else this.resolveReady();
	}

	private fail(error: Error): void {
		this.settleReady(error);
		for (const request of this.pending.values()) request.reject(new Error(GENERIC_READ_ERROR));
		this.pending.clear();
		this.retired = true;
		void this.worker.terminate();
	}

	private terminateWhenIdle(): void {
		if (this.retired && this.pending.size === 0) void this.worker.terminate();
	}
}

export class StaffCalendarSnapshot {
	private active: StaffCalendarWorkerClient | undefined;
	private refreshPromise: Promise<void> | undefined;
	private refreshTimer: ReturnType<typeof setInterval> | undefined;

	constructor(
		private endpoint: string,
		private createWorker: StaffCalendarWorkerFactory = (url) =>
			new NodeStaffCalendarWorkerClient(url),
		private refreshIntervalMs = DEFAULT_REFRESH_INTERVAL_MS
	) {}

	async initialize(): Promise<void> {
		if (!this.active) await this.refresh();
		if (!this.refreshTimer) {
			this.refreshTimer = setInterval(() => {
				void this.refresh();
			}, this.refreshIntervalMs);
			this.refreshTimer.unref();
		}
	}

	async getEventsBetween(start: Date, end: Date): Promise<EventInput[]> {
		await this.initialize();
		if (!this.active) throw new Error(GENERIC_READ_ERROR);
		return this.active.getEventsBetween(start, end);
	}

	async refresh(): Promise<void> {
		if (this.refreshPromise) return this.refreshPromise;

		const candidate = this.createWorker(this.endpoint);
		this.refreshPromise = (async () => {
			try {
				await candidate.ready();
				const previous = this.active;
				this.active = candidate;
				previous?.retire();
			} catch {
				candidate.retire();
				console.error(GENERIC_REFRESH_ERROR);
				if (!this.active) throw new Error(GENERIC_REFRESH_ERROR);
			} finally {
				this.refreshPromise = undefined;
			}
		})();

		return this.refreshPromise;
	}

	destroy(): void {
		if (this.refreshTimer) clearInterval(this.refreshTimer);
		this.refreshTimer = undefined;
		this.active?.retire();
		this.active = undefined;
	}
}

let staffCalendarSnapshot: StaffCalendarSnapshot | undefined;
let configuredEndpoint: string | undefined;

export function getStaffCalendarSnapshot(endpoint: string): StaffCalendarSnapshot {
	if (!staffCalendarSnapshot || configuredEndpoint !== endpoint) {
		staffCalendarSnapshot?.destroy();
		staffCalendarSnapshot = new StaffCalendarSnapshot(endpoint);
		configuredEndpoint = endpoint;
	}
	return staffCalendarSnapshot;
}

export async function initializeStaffCalendarSnapshot(endpoint: string): Promise<void> {
	await getStaffCalendarSnapshot(endpoint).initialize();
}
