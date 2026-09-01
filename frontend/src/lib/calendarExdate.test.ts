import { describe, expect, it } from 'vitest';
import { CalendarSet } from '$lib/calendar';

describe('calendar recurrence exclusions', () => {
	it('omits an EXDATE instance from a recurring event', () => {
		const calendar = CalendarSet.cleanAndParse(
			[
				'BEGIN:VCALENDAR',
				'VERSION:2.0',
				'PRODID:-//Makerspace//Calendar test//EN',
				'BEGIN:VEVENT',
				'UID:recurring-shift@example.com',
				'DTSTAMP:20260831T120000Z',
				'DTSTART:20260923T140000Z',
				'DTEND:20260923T150000Z',
				'RRULE:FREQ=DAILY;COUNT=3',
				'EXDATE:20260924T140000Z',
				'SUMMARY:Shira',
				'SEQUENCE:0',
				'END:VEVENT',
				'END:VCALENDAR',
				''
			].join('\r\n')
		);

		const events = calendar.between(
			new Date('2026-09-23T00:00:00Z'),
			new Date('2026-09-27T00:00:00Z')
		);

		expect(events.map((event) => event.start)).toEqual([
			'2026-09-23T14:00:00.000Z',
			'2026-09-25T14:00:00.000Z'
		]);
	});
});
