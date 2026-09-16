import { describe, expect, it } from 'vitest';
import type { Printer } from './prototype-data';
import { activityState, defaultFilters, matchesFilters, printerStates } from './printer-state';

const working: Printer = {
	id: 'arthur',
	name: 'Arthur',
	model: 'K1 Max',
	lifecycle: 'active',
	maintenance: 'none',
	condition: 'working',
	activity: 'idle',
	connected: true
};
const repair: Printer = {
	...working,
	id: 'bench',
	lifecycle: 'shelved',
	maintenance: 'repair',
	condition: 'out',
	location: 'Repair bench',
	activity: 'unknown',
	connected: false
};

describe('independent printer states', () => {
	it('can select shelved printers that are also broken and under repair', () => {
		const filters = {
			...defaultFilters,
			fleet: 'shelved' as const,
			maintenance: 'repair' as const,
			condition: 'out' as const
		};
		expect([working, repair].filter((p) => matchesFilters(p, filters)).map((p) => p.id)).toEqual([
			'bench'
		]);
		expect(matchesFilters(repair, defaultFilters)).toBe(false);
		expect(matchesFilters(repair, { ...filters, maintenance: 'testing' })).toBe(false);
	});
	it('treats idle as activity, not proof a printer is usable', () => {
		const brokenIdle = { ...working, condition: 'out' as const, maintenance: 'diagnosis' as const };
		expect(matchesFilters(brokenIdle, { ...defaultFilters, activity: 'idle' })).toBe(true);
		expect(
			matchesFilters(brokenIdle, { ...defaultFilters, activity: 'idle', condition: 'working' })
		).toBe(false);
	});
	it('does not erase a saved condition when live activity is missing', () => {
		expect(activityState(repair)).toBe('offline');
		expect(activityState({ ...working, stale: true })).toBe('unavailable');
		expect(
			matchesFilters({ ...working, stale: true }, { ...defaultFilters, activity: 'idle' })
		).toBe(false);
		expect(
			matchesFilters({ ...working, stale: true }, { ...defaultFilters, condition: 'working' })
		).toBe(true);
	});
	it('keeps older testing/repair records visible with maintenance separated', () => {
		expect(printerStates({ lifecycle: 'repair' })).toEqual({
			lifecycle: 'active',
			maintenance: 'repair'
		});
		expect(printerStates({ lifecycle: 'testing' })).toEqual({
			lifecycle: 'active',
			maintenance: 'testing'
		});
		expect(printerStates({ lifecycle: 'shelved', maintenance: 'testing' })).toEqual({
			lifecycle: 'shelved',
			maintenance: 'testing'
		});
	});
});
