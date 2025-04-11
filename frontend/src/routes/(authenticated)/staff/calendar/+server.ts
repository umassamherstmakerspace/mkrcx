import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';
import { CalendarServer } from '$lib/calendarServer';
import { generateColor } from '@marko19907/string-to-color';
import type { EventInput } from '@fullcalendar/core/index.js';

let calendar: CalendarServer | undefined = undefined;

const colorize = (event: EventInput) => {
	if (event.title === undefined) return event;

	return {
		...event,
		color: generateColor(event.title)
	};
};

export const GET: RequestHandler = async ({ fetch, url }) => {
	if (!calendar) {
		const cal = env.STAFF_CALENDAR_ENDPOINT;
		if (!cal) {
			error(500, 'No calendar endpoint configured.');
		}

		calendar = new CalendarServer(cal, fetch, colorize);
	}

	const start = url.searchParams.get('start');
	if (!start) error(400, 'No start date provided');

	const end = url.searchParams.get('end');
	if (!end) error(400, 'No end date provided');

	let data;

	try {
		data = await calendar.getEventsBetween(new Date(start || ''), new Date(end || ''));
	} catch (e) {
		calendar = undefined;
		console.error(e);
		error(500, {
			message: 'Internal Service Error'
		});
	}

	return new Response(JSON.stringify(data), {
		headers: {
			'Content-Type': 'text/calendar'
		}
	});
};
