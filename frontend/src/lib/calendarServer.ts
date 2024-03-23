import { error } from '@sveltejs/kit';
import { Cached } from '$lib/types';
import { CalendarSet } from '$lib/calendar';
import { type EventInput } from '@fullcalendar/core';

export class CalendarServer {
	private calendar: Cached<CalendarSet>;
	private colorize: (event: EventInput) => EventInput;

	constructor(
		url: string,
		fetch: typeof globalThis.fetch,
		colorize: (event: EventInput) => EventInput = (event: EventInput) => event
	) {
		this.calendar = new Cached(
			async () => {
				const req = await fetch(url);
				if (!req.ok) {
					error(500, 'Failed to fetch calendar.');
				}

				const calendarData = await req.text();
				return CalendarSet.cleanAndParse(calendarData);
			},
			1000 * 60 * 5
		); // 5 minutes
		this.colorize = colorize;
	}

	public async getEventsBetween(start: Date, end: Date): Promise<EventInput[]> {
		const data = await this.calendar.get();
		const events = data.between(start, end);

		return events.map(this.colorize);
	}
}
