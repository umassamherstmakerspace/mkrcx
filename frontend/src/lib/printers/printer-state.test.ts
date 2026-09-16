import { describe, expect, it } from 'vitest';
import type { Printer } from './prototype-data';
import { activityState, matchesFleetView, printerStates, printRecipient } from './printer-state';

const working: Printer = {
	id: 'arthur',
	name: 'Arthur',
	model: 'K1 Max',
	lifecycle: 'active',
	condition: 'working',
	activity: 'idle',
	connected: true
};
const shelved: Printer = {
	...working,
	id: 'bench',
	lifecycle: 'shelved',
	condition: 'out',
	activity: 'unknown',
	connected: false
};

describe('printer views', () => {
	it('distinguishes a recorded recipient from anonymous authorization placeholders', () => {
		expect(printRecipient('Fixture user')).toBe('Fixture user');
		for (const value of ['Staff override', 'Unlinked UCard', 'Unknown / unassigned', '', undefined])
			expect(printRecipient(value)).toBeUndefined();
	});
	it('surfaces an active machine error without changing the saved assessment', () => {
		const printer = { ...working, fault: 'Heater not heating' };
		expect(activityState(printer)).toBe('error');
		expect(matchesFleetView(printer, 'attention')).toBe(true);
		expect(printer.condition).toBe('working');
		expect(activityState({ ...printer, stale: true })).toBe('unavailable');
	});
	it('includes shelved problems in Needs attention, without treating offline as broken', () => {
		const printers = [
			working,
			shelved,
			{ ...working, id: 'limited', condition: 'limited' as const },
			{ ...working, id: 'offline', connected: false, activity: 'unknown' as const }
		];
		expect(printers.filter((p) => matchesFleetView(p, 'attention')).map((p) => p.id)).toEqual([
			'bench',
			'limited'
		]);
		expect(matchesFleetView({ ...shelved, condition: 'working' }, 'shelved')).toBe(true);
		expect(matchesFleetView({ ...shelved, condition: 'working' }, 'attention')).toBe(false);
	});
	it('keeps shelved printers out of In fleet and retired records out of every view', () => {
		expect(matchesFleetView(working, 'active')).toBe(true);
		expect(matchesFleetView(shelved, 'active')).toBe(false);
		for (const view of ['active', 'attention', 'shelved'] as const)
			expect(matchesFleetView({ ...shelved, lifecycle: 'retired' }, view)).toBe(false);
	});
	it('retains saved condition through missing live activity', () => {
		expect(activityState(shelved)).toBe('offline');
		expect(activityState({ ...working, stale: true })).toBe('unavailable');
		expect(matchesFleetView({ ...shelved, stale: true }, 'attention')).toBe(true);
	});
	it('preserves legacy records without making repair state a filter', () => {
		expect(printerStates({ lifecycle: 'repair' }).lifecycle).toBe('active');
		expect(
			matchesFleetView({ ...working, lifecycle: 'testing', maintenance: 'testing' }, 'active')
		).toBe(true);
		expect(matchesFleetView({ ...working, maintenance: 'diagnosis' }, 'attention')).toBe(false);
	});
});
