import type { Condition, FleetPlacement, Maintenance, Printer } from './prototype-data';

export const fleetLabels = { active: 'In fleet', shelved: 'Shelved', retired: 'Retired' };
export const maintenanceLabels = {
	none: 'None',
	diagnosis: 'Needs diagnosis',
	repair: 'In repair',
	testing: 'Testing'
};
export const conditionLabels = {
	working: 'Working',
	limited: 'Limited use',
	out: 'Out of service',
	unknown: 'Unknown'
};
export const activityLabels = {
	idle: 'Idle',
	printing: 'Printing',
	paused: 'Paused',
	offline: 'Offline',
	unavailable: 'Unavailable'
};

// Older snapshots combined fleet placement and maintenance in lifecycle.
export function printerStates(record: { lifecycle?: string; maintenance?: string }) {
	const legacy = record.lifecycle === 'testing' || record.lifecycle === 'repair';
	const lifecycle: FleetPlacement =
		record.lifecycle === 'shelved' || record.lifecycle === 'retired' ? record.lifecycle : 'active';
	const maintenance: Maintenance = legacy
		? (record.lifecycle as Maintenance)
		: ['diagnosis', 'repair', 'testing'].includes(record.maintenance ?? '')
			? (record.maintenance as Maintenance)
			: 'none';
	return { lifecycle, maintenance };
}

export function activityState(printer: Printer, disconnected = false): keyof typeof activityLabels {
	if (disconnected || printer.stale) return 'unavailable';
	if (printer.connected === false || printer.activity === 'unknown') return 'offline';
	return printer.activity;
}

export type FleetFilters = {
	fleet: FleetPlacement | 'all';
	condition: Condition | 'all';
	maintenance: Maintenance | 'all';
	activity: keyof typeof activityLabels | 'all';
};
export const defaultFilters: FleetFilters = {
	fleet: 'active',
	condition: 'all',
	maintenance: 'all',
	activity: 'all'
};

export function matchesFilters(
	printer: Printer,
	filters: FleetFilters,
	disconnected = false
): boolean {
	const state = printerStates(printer);
	return (
		(filters.fleet === 'all' || state.lifecycle === filters.fleet) &&
		(filters.condition === 'all' || printer.condition === filters.condition) &&
		(filters.maintenance === 'all' || state.maintenance === filters.maintenance) &&
		(filters.activity === 'all' || activityState(printer, disconnected) === filters.activity)
	);
}
