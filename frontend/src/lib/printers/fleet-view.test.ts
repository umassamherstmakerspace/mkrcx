import { describe, expect, it } from 'vitest';
import { printers } from './prototype-data';
import { remainingMinutes, sortFleet } from './fleet-view';

describe('printer fleet ordering', () => {
	it('groups status first, active before idle, then Max before K1/K1C', () => {
		expect(sortFleet(printers).map((printer) => printer.name)).toEqual([
			'Matthew Barney',
			'Tim Hawkinson',
			'Cai Guo-Qiang',
			'Richard Serra',
			'Simone Leigh',
			'Douglas Tilden',
			'George Rickey',
			'Doris Salcedo',
			'Jean Tinguely',
			'Louise Bourgeois',
			'Julie Mehretu',
			'Arthur Ganson',
			'Augusta Savage',
			'Gabriela Salazar',
			'Laurie Anderson',
			'Alexander Calder'
		]);
	});
	it('allows another column and reverse order without changing the source fleet', () => {
		const original = printers.map((printer) => printer.id);
		expect(sortFleet(printers, 'name', 'asc')[0].name).toBe('Alexander Calder');
		expect(sortFleet(printers, 'name', 'desc')[0].name).toBe('Tim Hawkinson');
		expect(sortFleet(printers, 'condition', 'desc')[0].name).toBe('Laurie Anderson');
		expect(printers.map((printer) => printer.id)).toEqual(original);
	});
	it('sorts known estimates numerically and keeps unavailable estimates last in both directions', () => {
		expect(
			sortFleet(printers, 'remaining', 'asc')
				.slice(0, 4)
				.map((printer) => printer.minutes)
		).toEqual([18, 42, 105, 190]);
		expect(
			sortFleet(printers, 'remaining', 'desc')
				.slice(0, 4)
				.map((printer) => printer.minutes)
		).toEqual([190, 105, 42, 18]);
		for (const direction of ['asc', 'desc'] as const) {
			expect(
				sortFleet(printers, 'remaining', direction)
					.slice(4)
					.every((printer) => remainingMinutes(printer, false) === null)
			).toBe(true);
		}
	});
	it('does not use stale or disconnected estimates as live sorting data', () => {
		const active = printers.find((printer) => printer.name === 'Simone Leigh')!;
		expect(remainingMinutes({ ...active, stale: true }, false)).toBeNull();
		expect(remainingMinutes(active, true)).toBeNull();
		const mixed = [
			{ ...active, stale: true },
			printers.find((printer) => printer.name === 'Richard Serra')!
		];
		expect(sortFleet(mixed, 'remaining')[0].name).toBe('Richard Serra');
	});
});
