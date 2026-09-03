/** Static fleet identity. Runtime status and print data arrive from the printer collector. */
export type Condition = 'working' | 'limited' | 'out' | 'unknown';
export type Activity = 'idle' | 'printing' | 'paused' | 'unknown';
export type Printer = {
	id: string;
	machineId?: string;
	name: string;
	model: 'K1' | 'K1C' | 'K1 Max';
	condition: Condition;
	activity: Activity;
	note?: string;
	minutes?: number;
	progress?: number;
	stale?: boolean;
	job?: { person: string; file: string; material: string; started: string };
};

const fleet: Array<Pick<Printer, 'id' | 'machineId' | 'name' | 'model'>> = [
	{ id: 'k1-30eb', machineId: 'K1-30EB', name: 'Richard Serra', model: 'K1' },
	{ id: 'k1-32b8', machineId: 'K1-32B8', name: 'Louise Bourgeois', model: 'K1' },
	{ id: 'k1-69ba', machineId: 'K1-69BA', name: 'Julie Mehretu', model: 'K1' },
	{ id: 'k1-791a', machineId: 'K1-791A', name: 'Cai Guo-Qiang', model: 'K1' },
	{ id: 'k1c-1c58', machineId: 'K1C-1C58', name: 'Jean Tinguely', model: 'K1C' },
	{ id: 'k1c-1d56', machineId: 'K1C-1D56', name: 'Alexander Calder', model: 'K1C' },
	{ id: 'k1c-1f44', machineId: 'K1C-1F44', name: 'Doris Salcedo', model: 'K1C' },
	{ id: 'k1c-1f65', machineId: 'K1C-1F65', name: 'Gabriela Salazar', model: 'K1C' },
	{ id: 'k1c-1f94', machineId: 'K1C-1F94', name: 'Simone Leigh', model: 'K1C' },
	{ id: 'k1max-2a33', machineId: 'K1MAX-2A33', name: 'Matthew Barney', model: 'K1 Max' },
	{ id: 'k1max-d101', machineId: 'K1MAX-D101', name: 'Arthur Ganson', model: 'K1 Max' },
	{ id: 'k1max-d103', machineId: 'K1MAX-D103', name: 'George Rickey', model: 'K1 Max' },
	{ id: 'k1max-d949', machineId: 'K1MAX-D949', name: 'Douglas Tilden', model: 'K1 Max' },
	{ id: 'k1max-d973', machineId: 'K1MAX-D973', name: 'Augusta Savage', model: 'K1 Max' },
	{ id: 'k1max-fb47', machineId: 'K1MAX-FB47', name: 'Tim Hawkinson', model: 'K1 Max' },
	{ id: 'fdm-k1max-anderson', name: 'Laurie Anderson', model: 'K1 Max' }
];

export const printers: Printer[] = fleet.map((printer) => ({
	...printer,
	condition: 'unknown',
	activity: 'unknown',
	stale: true
}));

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
