/** Design fixtures only. Every condition, activity, note and person below is invented. */
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

export const printers: Printer[] = [
	{
		id: '1F44',
		machineId: 'K1C-1F44',
		name: 'Doris Salcedo',
		model: 'K1C',
		condition: 'working',
		activity: 'idle'
	},
	{
		id: '1F94',
		machineId: 'K1C-1F94',
		name: 'Simone Leigh',
		model: 'K1C',
		condition: 'working',
		activity: 'printing',
		minutes: 42,
		progress: 68,
		job: {
			person: 'Alex Morgan',
			file: 'enclosure_v3.3mf',
			material: 'PLA · Natural',
			started: '1:10 PM'
		}
	},
	{
		id: 'D101',
		machineId: 'K1MAX-D101',
		name: 'Arthur Ganson',
		model: 'K1 Max',
		condition: 'limited',
		activity: 'idle',
		note: 'PLA only. Please use another printer for PETG or ABS.'
	},
	{
		id: '1F65',
		machineId: 'K1C-1F65',
		name: 'Gabriela Salazar',
		model: 'K1C',
		condition: 'out',
		activity: 'idle',
		note: 'Extruder jam. Waiting for a replacement part.'
	},
	{
		id: '30EB',
		machineId: 'K1-30EB',
		name: 'Richard Serra',
		model: 'K1',
		condition: 'working',
		activity: 'printing',
		minutes: 105,
		progress: 35,
		job: {
			person: 'Jordan Chen',
			file: 'gearbox_housing.3mf',
			material: 'PETG · Black',
			started: '1:05 PM'
		}
	},
	{
		id: '32B8',
		machineId: 'K1-32B8',
		name: 'Louise Bourgeois',
		model: 'K1',
		condition: 'working',
		activity: 'idle'
	},
	{
		id: '69BA',
		machineId: 'K1-69BA',
		name: 'Julie Mehretu',
		model: 'K1',
		condition: 'limited',
		activity: 'printing',
		minutes: 18,
		progress: 89,
		note: 'PLA only.',
		job: {
			person: 'Sam Patel',
			file: 'mounting_bracket.3mf',
			material: 'PLA · Blue',
			started: '12:20 PM'
		}
	},
	{
		id: '791A',
		machineId: 'K1-791A',
		name: 'Cai Guo-Qiang',
		model: 'K1',
		condition: 'working',
		activity: 'paused',
		progress: 51,
		job: {
			person: 'Taylor Brooks',
			file: 'lamp_base.3mf',
			material: 'PLA · White',
			started: '12:40 PM'
		}
	},
	{
		id: '1C58',
		machineId: 'K1C-1C58',
		name: 'Jean Tinguely',
		model: 'K1C',
		condition: 'working',
		activity: 'idle'
	},
	{
		id: '1D56',
		machineId: 'K1C-1D56',
		name: 'Alexander Calder',
		model: 'K1C',
		condition: 'unknown',
		activity: 'unknown',
		stale: true
	},
	{
		id: '2A33',
		machineId: 'K1MAX-2A33',
		name: 'Matthew Barney',
		model: 'K1 Max',
		condition: 'working',
		activity: 'printing',
		minutes: 190,
		progress: 22,
		job: {
			person: 'Unknown / unassigned',
			file: 'display_stand.3mf',
			material: 'PLA · Gray',
			started: '12:55 PM'
		}
	},
	{
		id: 'D103',
		machineId: 'K1MAX-D103',
		name: 'George Rickey',
		model: 'K1 Max',
		condition: 'working',
		activity: 'idle'
	},
	{
		id: 'D949',
		machineId: 'K1MAX-D949',
		name: 'Douglas Tilden',
		model: 'K1 Max',
		condition: 'working',
		activity: 'idle'
	},
	{
		id: 'fdm-k1max-anderson',
		name: 'Laurie Anderson',
		model: 'K1 Max',
		condition: 'unknown',
		activity: 'unknown'
	},
	{
		id: 'D973',
		machineId: 'K1MAX-D973',
		name: 'Augusta Savage',
		model: 'K1 Max',
		condition: 'out',
		activity: 'idle',
		note: 'Bed heater needs repair.'
	},
	{
		id: 'FB47',
		machineId: 'K1MAX-FB47',
		name: 'Tim Hawkinson',
		model: 'K1 Max',
		condition: 'working',
		activity: 'printing',
		minutes: undefined,
		progress: 4,
		job: {
			person: 'Casey Rivera',
			file: 'sculpture_study.3mf',
			material: 'PLA · Red',
			started: '1:57 PM'
		}
	}
];

export function duration(minutes: number): string {
	if (minutes < 60) return `${minutes} min`;
	const hours = Math.floor(minutes / 60);
	const remainder = minutes % 60;
	return `${hours} hr${remainder ? ` ${remainder} min` : ''}`;
}

// All fixtures share a deliberately fixed 2:00 PM snapshot, not the wall clock.
export function finishTime(minutes: number): string {
	const total = 14 * 60 + minutes;
	const hour = Math.floor(total / 60) % 24;
	return `${hour % 12 || 12}:${String(total % 60).padStart(2, '0')} ${hour >= 12 ? 'PM' : 'AM'}`;
}
