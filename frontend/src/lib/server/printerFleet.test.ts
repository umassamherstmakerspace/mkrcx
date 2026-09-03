import { beforeEach, describe, expect, it } from 'vitest';
import { printers } from '$lib/printers/prototype-data';
import { acceptSnapshot, fleetResponse, resetForTest, tokenMatches } from './printerFleet';

const now = Date.parse('2026-09-03T17:00:00Z');
function snapshot() {
	return {
		fetchedAt: new Date(now).toISOString(),
		printers: printers.map(({ id }) => ({
			id,
			condition: 'working' as const,
			activity: 'idle' as const,
			job:
				id === 'k1c-1f44'
					? { person: 'Student Name', file: 'part.gcode', material: 'PLA', started: '12:30 PM' }
					: undefined
		}))
	};
}

describe('printer fleet server boundary', () => {
	beforeEach(resetForTest);
	it('checks collector bearer credentials', () => {
		expect(tokenMatches('Bearer correct', 'correct')).toBe(true);
		expect(tokenMatches('Bearer wrong', 'correct')).toBe(false);
		expect(tokenMatches(null, 'correct')).toBe(false);
	});
	it('accepts only a complete, current, uniquely identified fleet', () => {
		expect(() => acceptSnapshot(snapshot(), now)).not.toThrow();
		expect(() =>
			acceptSnapshot({ ...snapshot(), printers: snapshot().printers.slice(1) }, now)
		).toThrow();
		expect(() =>
			acceptSnapshot({ ...snapshot(), fetchedAt: '2026-09-03T16:00:00Z' }, now)
		).toThrow();
	});
	it('removes job details from public data and retains them for staff', () => {
		acceptSnapshot(snapshot(), now);
		expect(fleetResponse(false, now).printers.some(({ job }) => job)).toBe(false);
		expect(fleetResponse(true, now).printers.find(({ id }) => id === 'k1c-1f44')?.job?.person).toBe(
			'Student Name'
		);
	});
	it('fails stale readings closed', () => {
		acceptSnapshot(snapshot(), now);
		const result = fleetResponse(true, now + 90_001);
		expect(result.stale).toBe(true);
		expect(
			result.printers.every(
				({ condition, activity, job }) => condition === 'unknown' && activity === 'unknown' && !job
			)
		).toBe(true);
	});
});
