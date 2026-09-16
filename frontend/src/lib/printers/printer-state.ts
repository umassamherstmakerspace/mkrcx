import type { FleetPlacement, Maintenance, Printer } from './prototype-data';

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
	out: 'Broken',
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

export const fleetViews = [
	{ id: 'active', label: 'In fleet' },
	{ id: 'attention', label: 'Needs attention' },
	{ id: 'shelved', label: 'Shelved' }
] as const;
export type FleetView = (typeof fleetViews)[number]['id'];

export function matchesFleetView(printer: Printer, view: FleetView): boolean {
	const { lifecycle } = printerStates(printer);
	if (lifecycle === 'retired') return false;
	if (view === 'attention') return printer.condition === 'limited' || printer.condition === 'out';
	return lifecycle === view;
}
