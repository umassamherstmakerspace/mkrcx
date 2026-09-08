/** Public printer view. Identities and staff records are loaded from the backend registry. */
export type Condition = 'working' | 'limited' | 'out' | 'unknown';
export type Activity = 'idle' | 'printing' | 'paused' | 'unknown';
export type Printer = {
	id: string;
	machineId?: string;
	name: string;
	model: string;
	lifecycle?: 'active' | 'testing' | 'repair' | 'retired';
	location?: string;
	conditionSource?: 'record' | 'station';
	conditionUpdatedAt?: string | null;
	lastSeen?: string | null;
	connected?: boolean;
	condition: Condition;
	activity: Activity;
	note?: string;
	minutes?: number;
	progress?: number;
	stale?: boolean;
	job?: { person: string; file: string; material: string; started: string };
};

export function duration(minutes: number): string {
	if (minutes < 60) return `${minutes} min`;
	const hours = Math.floor(minutes / 60);
	const remainder = minutes % 60;
	return `${hours} hr${remainder ? ` ${remainder} min` : ''}`;
}

export function finishTime(minutes: number): string {
	return new Date(Date.now() + minutes * 60_000).toLocaleTimeString([], {
		hour: 'numeric',
		minute: '2-digit'
	});
}
