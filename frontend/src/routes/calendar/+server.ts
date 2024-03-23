import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';
import { CalendarServer } from '$lib/calendarServer';
import type { EventInput } from '@fullcalendar/core/index.js';

let calendar: CalendarServer;

const colorize = (event: EventInput) => {
	if (event.title === undefined) return event;

	const title = event.title.toLocaleLowerCase();

	if (title.includes('closed')) {
		return {
			...event,
			color: '#820318'
		};
	} else if (title.includes('open')) {
		return {
			...event,
			color: '#c78c20'
		};
	}

	return event;
};

export const GET: RequestHandler = async ({ fetch, url }) => {
	if (!calendar) {
		const cal = env.CALENDAR_ENDPOINT;
		if (!cal) {
			error(500, 'No calendar endpoint configured.');
		}

		calendar = new CalendarServer(cal, fetch, colorize);
	}

	const start = url.searchParams.get('start');
	if (!start) error(400, 'No start date provided');

	const end = url.searchParams.get('end');
	if (!end) error(400, 'No end date provided');

	const data = await calendar.getEventsBetween(new Date(start || ''), new Date(end || ''));

	return new Response(JSON.stringify(data), {
		headers: {
			'Content-Type': 'text/calendar'
		}
	});
};
