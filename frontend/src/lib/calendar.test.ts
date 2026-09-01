import { describe, expect, it } from 'vitest';
import { CalendarSet } from '$lib/calendar';

describe('calendar event details', () => {
	it('preserves the description and location from an ICS event', () => {
		const calendar = CalendarSet.cleanAndParse(
			[
				'BEGIN:VCALENDAR',
				'VERSION:2.0',
				'PRODID:-//Makerspace//Calendar test//EN',
				'BEGIN:VEVENT',
				'UID:tour-test@example.com',
				'DTSTAMP:20260831T120000Z',
				'DTSTART:20260923T140000Z',
				'DTEND:20260923T150000Z',
				'SUMMARY:Prospective student tour',
				'DESCRIPTION:Meet the group at the front desk.',
				'LOCATION:Agricultural Engineering 120',
				'SEQUENCE:0',
				'END:VEVENT',
				'END:VCALENDAR',
				''
			].join('\r\n')
		);

		const events = calendar.between(
			new Date('2026-09-23T00:00:00Z'),
			new Date('2026-09-24T00:00:00Z')
		);

		expect(events).toHaveLength(1);
		expect(events[0]).toMatchObject({
			title: 'Prospective student tour',
			description: 'Meet the group at the front desk.',
			location: 'Agricultural Engineering 120'
		});
	});

	it('decodes escaped text in event details', () => {
		const calendar = CalendarSet.cleanAndParse(
			[
				'BEGIN:VCALENDAR',
				'VERSION:2.0',
				'PRODID:-//Makerspace//Calendar test//EN',
				'BEGIN:VEVENT',
				'UID:cold-storage@example.com',
				'DTSTAMP:20260831T120000Z',
				'DTSTART:20260923T140000Z',
				'DTEND:20260923T150000Z',
				'SUMMARY:Cold storage\\, phase 1',
				'DESCRIPTION:Bring back chairs\\, tables\\; and tools.\\n\\nMeet in Chenoweth 116.',
				'LOCATION:Chenoweth 116\\, UMass',
				'SEQUENCE:0',
				'END:VEVENT',
				'END:VCALENDAR',
				''
			].join('\r\n')
		);

		const events = calendar.between(
			new Date('2026-09-23T00:00:00Z'),
			new Date('2026-09-24T00:00:00Z')
		);

		expect(events[0]).toMatchObject({
			title: 'Cold storage, phase 1',
			description: 'Bring back chairs, tables; and tools.\n\nMeet in Chenoweth 116.',
			location: 'Chenoweth 116, UMass'
		});
	});

	it('omits excluded instances from a recurring event', () => {
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
