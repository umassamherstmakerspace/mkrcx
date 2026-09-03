import { timingSafeEqual } from 'node:crypto';
import {
	printers as fleet,
	type Activity,
	type Condition,
	type Printer
} from '$lib/printers/prototype-data';

const MAX_AGE_MS = 90_000;
const allowedIds = new Set(fleet.map(({ id }) => id));
export type CollectorPrinter = {
	id: string;
	condition: Condition;
	activity: Activity;
	note?: string;
	minutes?: number;
	progress?: number;
	job?: { person: string; file: string; material: string; started: string };
};
export type CollectorSnapshot = { fetchedAt: string; printers: CollectorPrinter[] };
let latest: { receivedAt: number; snapshot: CollectorSnapshot } | undefined;

export function tokenMatches(supplied: string | null, expected: string | undefined): boolean {
	if (!expected || !supplied?.startsWith('Bearer ')) return false;
	const actual = Buffer.from(supplied.slice(7));
	const wanted = Buffer.from(expected);
	return actual.length === wanted.length && timingSafeEqual(actual, wanted);
}

export function acceptSnapshot(value: unknown, now = Date.now()): void {
	if (!value || typeof value !== 'object') throw new TypeError('Invalid snapshot.');
	const input = value as Partial<CollectorSnapshot>;
	if (!Array.isArray(input.printers) || input.printers.length !== fleet.length)
		throw new TypeError('Snapshot must contain the complete fleet.');
	const fetched = Date.parse(String(input.fetchedAt ?? ''));
	if (!Number.isFinite(fetched) || Math.abs(now - fetched) > MAX_AGE_MS)
		throw new TypeError('Snapshot timestamp is stale.');
	const seen = new Set<string>();
	for (const printer of input.printers) {
		if (
			!printer ||
			typeof printer !== 'object' ||
			!allowedIds.has(printer.id) ||
			seen.has(printer.id)
		)
			throw new TypeError('Snapshot printer identity is invalid.');
		seen.add(printer.id);
		if (!['working', 'limited', 'out', 'unknown'].includes(printer.condition))
			throw new TypeError('Invalid condition.');
		if (!['idle', 'printing', 'paused', 'unknown'].includes(printer.activity))
			throw new TypeError('Invalid activity.');
		if ((printer.note?.length ?? 0) > 2000) throw new TypeError('Note is too long.');
	}
	latest = { receivedAt: now, snapshot: input as CollectorSnapshot };
}

export function fleetResponse(staff: boolean, now = Date.now()) {
	const stale = !latest || now - latest.receivedAt > MAX_AGE_MS;
	const readings = new Map(
		(latest?.snapshot.printers ?? []).map((printer) => [printer.id, printer])
	);
	const printers: Printer[] = fleet.map((identity) => {
		const reading = readings.get(identity.id);
		if (stale || !reading)
			return { ...identity, condition: 'unknown', activity: 'unknown', stale: true };
		const publicFields: Printer = {
			...identity,
			condition: reading.condition,
			activity: reading.activity,
			note: reading.note,
			minutes: reading.minutes,
			progress: reading.progress,
			stale: false
		};
		return staff && reading.job ? { ...publicFields, job: reading.job } : publicFields;
	});
	return {
		audience: staff ? 'staff' : 'public',
		stale,
		fetchedAt: latest?.snapshot.fetchedAt ?? null,
		printers
	};
}

export function resetForTest() {
	latest = undefined;
}
