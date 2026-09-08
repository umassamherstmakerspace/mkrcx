import { describe, expect, it } from 'vitest';
import { type Printer } from './prototype-data';
import { remainingMinutes, sortFleet } from './fleet-view';

const printer = (
	name: string,
	condition: Printer['condition'],
	activity: Printer['activity'],
	model: Printer['model'] = 'K1'
): Printer => ({ id: name, name, model, condition, activity });

describe('printer fleet ordering', () => {
	it('sorts newly registered printer models without a code change', () => {
		expect(
			sortFleet([
				printer('new', 'working', 'idle', 'New model'),
				printer('max', 'working', 'idle', 'K1 Max')
			]).map((p) => p.name)
		).toEqual(['max', 'new']);
	});
	it('groups status first, active before idle, then Max before K1/K1C', () => {
		const input = [
			printer('red', 'out', 'idle'),
			printer('idle', 'working', 'idle'),
			printer('active', 'working', 'printing', 'K1 Max'),
			printer('yellow', 'limited', 'idle')
		];
		expect(sortFleet(input).map(({ name }) => name)).toEqual(['active', 'idle', 'yellow', 'red']);
	});
	it('keeps missing estimates last in either direction without changing the source', () => {
		const input = [
			{ ...printer('short', 'working', 'printing'), minutes: 10 },
			{ ...printer('long', 'working', 'printing'), minutes: 40 },
			printer('idle', 'working', 'idle')
		];
		expect(sortFleet(input, 'remaining', 'asc').map(({ name }) => name)).toEqual([
			'short',
			'long',
			'idle'
		]);
		expect(sortFleet(input, 'remaining', 'desc').map(({ name }) => name)).toEqual([
			'long',
			'short',
			'idle'
		]);
		expect(remainingMinutes(input[2], false)).toBeNull();
	});
});
