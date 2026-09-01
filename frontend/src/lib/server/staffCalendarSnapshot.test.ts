import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { EventInput } from '@fullcalendar/core';
import {
	StaffCalendarSnapshot,
	type StaffCalendarWorkerClient
} from '$lib/server/staffCalendarSnapshot';

function deferred() {
	let resolve!: () => void;
	let reject!: (reason: Error) => void;
	const promise = new Promise<void>((resolvePromise, rejectPromise) => {
		resolve = resolvePromise;
		reject = rejectPromise;
	});
	return { promise, resolve, reject };
}

class FakeWorker implements StaffCalendarWorkerClient {
	readonly readyGate = deferred();
	retired = false;
	events: EventInput[] = [];
	getEventsBetween = vi.fn(async () => this.events);

	ready(): Promise<void> {
		return this.readyGate.promise;
	}

	retire(): void {
		this.retired = true;
	}
}

describe('StaffCalendarSnapshot', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.restoreAllMocks();
	});

	it('prewarms one worker and serves its parsed snapshot', async () => {
		const worker = new FakeWorker();
		const factory = vi.fn(() => worker);
		const snapshot = new StaffCalendarSnapshot('private calendar URL', factory);
		const initialize = snapshot.initialize();

		expect(factory).toHaveBeenCalledOnce();
		expect(factory).toHaveBeenCalledWith('private calendar URL');
		worker.readyGate.resolve();
		await initialize;

		await snapshot.getEventsBetween(new Date(0), new Date(1));
		expect(worker.getEventsBetween).toHaveBeenCalledOnce();
		snapshot.destroy();
		expect(worker.retired).toBe(true);
	});

	it('continues serving the last good snapshot while a replacement warms', async () => {
		const first = new FakeWorker();
		const replacement = new FakeWorker();
		first.events = [{ title: 'old snapshot' }];
		replacement.events = [{ title: 'new snapshot' }];
		const workers = [first, replacement];
		const snapshot = new StaffCalendarSnapshot('private calendar URL', () => workers.shift()!);

		const initialize = snapshot.initialize();
		first.readyGate.resolve();
		await initialize;

		const refresh = snapshot.refresh();
		await expect(snapshot.getEventsBetween(new Date(0), new Date(1))).resolves.toEqual([
			{ title: 'old snapshot' }
		]);
		expect(first.retired).toBe(false);

		replacement.readyGate.resolve();
		await refresh;
		expect(first.retired).toBe(true);
		await expect(snapshot.getEventsBetween(new Date(0), new Date(1))).resolves.toEqual([
			{ title: 'new snapshot' }
		]);
		snapshot.destroy();
	});

	it('deduplicates overlapping refresh attempts', async () => {
		const first = new FakeWorker();
		const replacement = new FakeWorker();
		const factory = vi.fn().mockReturnValueOnce(first).mockReturnValueOnce(replacement);
		const snapshot = new StaffCalendarSnapshot('private calendar URL', factory);

		const initialize = snapshot.initialize();
		first.readyGate.resolve();
		await initialize;

		const refreshOne = snapshot.refresh();
		const refreshTwo = snapshot.refresh();
		expect(factory).toHaveBeenCalledTimes(2);
		replacement.readyGate.resolve();
		await Promise.all([refreshOne, refreshTwo]);
		expect(first.retired).toBe(true);
		snapshot.destroy();
	});

	it('retains the last good snapshot when a refresh fails without logging the private URL', async () => {
		const first = new FakeWorker();
		const failed = new FakeWorker();
		first.events = [{ title: 'last good snapshot' }];
		const workers = [first, failed];
		const error = vi.spyOn(console, 'error').mockImplementation(() => undefined);
		const snapshot = new StaffCalendarSnapshot('private calendar URL', () => workers.shift()!);

		const initialize = snapshot.initialize();
		first.readyGate.resolve();
		await initialize;

		const refresh = snapshot.refresh();
		failed.readyGate.reject(new Error('failure mentioning private calendar URL'));
		await refresh;

		expect(failed.retired).toBe(true);
		expect(first.retired).toBe(false);
		expect(error).toHaveBeenCalledWith('Failed to refresh staff calendar snapshot.');
		expect(JSON.stringify(error.mock.calls)).not.toContain('private calendar URL');
		await expect(snapshot.getEventsBetween(new Date(0), new Date(1))).resolves.toEqual([
			{ title: 'last good snapshot' }
		]);
		snapshot.destroy();
	});

	it('refreshes on a ten-minute timer without replacing the active snapshot early', async () => {
		const first = new FakeWorker();
		const replacement = new FakeWorker();
		const factory = vi.fn().mockReturnValueOnce(first).mockReturnValueOnce(replacement);
		const snapshot = new StaffCalendarSnapshot('private calendar URL', factory, 10 * 60 * 1000);

		const initialize = snapshot.initialize();
		first.readyGate.resolve();
		await initialize;

		await vi.advanceTimersByTimeAsync(10 * 60 * 1000);
		expect(factory).toHaveBeenCalledTimes(2);
		expect(first.retired).toBe(false);
		replacement.readyGate.resolve();
		await vi.runAllTicks();
		expect(first.retired).toBe(true);
		snapshot.destroy();
	});
});
