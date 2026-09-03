import type { Printer } from './prototype-data';

export type SortKey = 'name' | 'model' | 'condition' | 'activity' | 'remaining';
export type SortDirection = 'asc' | 'desc';

const conditionRank = { working: 0, limited: 1, out: 2, unknown: 3 };
const modelRank = { 'K1 Max': 0, K1: 1, K1C: 1 };

function activityRank(printer: Printer, disconnected: boolean) {
	if (disconnected || printer.stale || printer.activity === 'unknown') return 2;
	return printer.activity === 'idle' ? 1 : 0;
}

export function remainingMinutes(printer: Printer, disconnected: boolean): number | null {
	return !disconnected &&
		!printer.stale &&
		printer.activity === 'printing' &&
		typeof printer.minutes === 'number' &&
		Number.isFinite(printer.minutes)
		? printer.minutes
		: null;
}

function defaultOrder(a: Printer, b: Printer, disconnected: boolean): number {
	return (
		conditionRank[a.condition] - conditionRank[b.condition] ||
		activityRank(a, disconnected) - activityRank(b, disconnected) ||
		modelRank[a.model] - modelRank[b.model] ||
		a.name.localeCompare(b.name) ||
		a.id.localeCompare(b.id)
	);
}

export function sortFleet(
	printers: readonly Printer[],
	key: SortKey = 'condition',
	direction: SortDirection = 'asc',
	disconnected = false
): Printer[] {
	return [...printers].sort((a, b) => {
		let primary = 0;
		if (key === 'name') primary = a.name.localeCompare(b.name);
		if (key === 'model') primary = modelRank[a.model] - modelRank[b.model];
		if (key === 'condition') primary = conditionRank[a.condition] - conditionRank[b.condition];
		if (key === 'activity') primary = activityRank(a, disconnected) - activityRank(b, disconnected);
		if (key === 'remaining') {
			const left = remainingMinutes(a, disconnected);
			const right = remainingMinutes(b, disconnected);
			// Missing or stale estimates stay at the bottom in either direction.
			if (left === null && right !== null) return 1;
			if (right === null && left !== null) return -1;
			primary = left === null || right === null ? 0 : left - right;
		}
		return primary * (direction === 'asc' ? 1 : -1) || defaultOrder(a, b, disconnected);
	});
}
