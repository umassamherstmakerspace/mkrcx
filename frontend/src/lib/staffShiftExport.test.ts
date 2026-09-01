import { describe, expect, it } from 'vitest';
import {
	buildStaffShiftCalendar,
	escapeIcsText,
	foldIcsLine,
	resolveStaffShiftName,
	staffShiftEvents,
	staffShiftFilename
} from '$lib/staffShiftExport';

describe('staff shift export', () => {
	it('maps the account first name to the maintained staff roster', () => {
		expect(resolveStaffShiftName({ email: 'shira@example.edu', name: 'Shira Epstein' })).toBe(
			'Shira'
		);
	});

	it('supports a lower-case email override for exceptional account names', () => {
		expect(
			resolveStaffShiftName(
				{ email: 'Alias@Example.edu', name: 'Different Name' },
				{ 'alias@example.edu': 'Niall' }
			)
		).toBe('Niall');
	});

	it('selects exact timed shifts without including other events', () => {
		const events = [
			{ id: 'one', title: ' Shira ', start: '2026-09-01T13:00:00Z', end: '2026-09-01T17:00:00Z' },
			{
				id: 'two',
				title: 'Shira meeting',
				start: '2026-09-02T13:00:00Z',
				end: '2026-09-02T14:00:00Z'
			},
			{ id: 'three', title: 'Lauren', start: '2026-09-03T13:00:00Z', end: '2026-09-03T17:00:00Z' },
			{ id: 'four', title: 'Shira', allDay: true, start: '2026-09-04', end: '2026-09-05' }
		];

		expect(staffShiftEvents(events, 'Shira')).toEqual([events[0]]);
	});

	it('creates a standards-shaped calendar with stable occurrence IDs', () => {
		const result = buildStaffShiftCalendar(
			'Shira',
			[
				{
					id: 'source-event',
					title: 'Shira',
					start: '2026-09-01T13:00:00Z',
					end: '2026-09-01T17:00:00Z'
				}
			],
			new Date('2026-08-31T12:34:56Z')
		);

		expect(result).toContain('BEGIN:VCALENDAR\r\n');
		expect(result).toContain('UID:source-event-20260901T130000Z@mkr.cx\r\n');
		expect(result).toContain('DTSTAMP:20260831T123456Z\r\n');
		expect(result).toContain('DTSTART:20260901T130000Z\r\n');
		expect(result).toContain('DTEND:20260901T170000Z\r\n');
		expect(result.match(/BEGIN:VEVENT/g)).toHaveLength(1);
		expect(result.endsWith('END:VCALENDAR\r\n')).toBe(true);
	});

	it('escapes text, folds UTF-8 lines, and produces a safe filename', () => {
		expect(escapeIcsText('one, two; three\nfour\\five')).toBe('one\\, two\\; three\\nfour\\\\five');
		const folded = foldIcsLine(`DESCRIPTION:${'é'.repeat(80)}`);
		expect(folded).toContain('\r\n ');
		for (const line of folded.split('\r\n')) {
			expect(new TextEncoder().encode(line).length).toBeLessThanOrEqual(75);
		}
		expect(staffShiftFilename('Shira')).toBe('makerspace-shira-shifts.ics');
	});
});
