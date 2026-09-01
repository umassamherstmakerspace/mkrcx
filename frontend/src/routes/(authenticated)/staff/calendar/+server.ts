import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { env as privateEnv } from '$env/dynamic/private';
import { env as publicEnv } from '$env/dynamic/public';
import type { CalendarServer } from '$lib/calendarServer';
import { colorizeStaffCalendarEvent } from '$lib/staffCalendarColors';
import { getStaffCalendarSnapshot } from '$lib/server/staffCalendarSnapshot';
import {
	requireStaffCalendarAccess,
	type StaffCalendarAccessOptions
} from '$lib/server/staffCalendarAccess';

interface CalendarReader {
	getEventsBetween(start: Date, end: Date): ReturnType<CalendarServer['getEventsBetween']>;
}

interface StaffCalendarHandlerContext {
	fetch: typeof globalThis.fetch;
	url: URL;
	cookies: { get(name: string): string | undefined };
}

interface StaffCalendarHandlerDependencies {
	getCalendarEndpoint: () => string | undefined;
	getLeashEndpoint: () => string | undefined;
	authorize: (options: StaffCalendarAccessOptions) => Promise<unknown>;
	createCalendar: (endpoint: string, fetch: typeof globalThis.fetch) => CalendarReader;
}

export function _createStaffCalendarHandler(
	dependencies: StaffCalendarHandlerDependencies
): (context: StaffCalendarHandlerContext) => Promise<Response> {
	let calendar: CalendarReader | undefined;

	return async ({ fetch, url, cookies }) => {
		await dependencies.authorize({
			token: cookies.get('token'),
			leashURL: dependencies.getLeashEndpoint(),
			fetch
		});

		const startParam = url.searchParams.get('start');
		if (!startParam) error(400, 'No start date provided');

		const endParam = url.searchParams.get('end');
		if (!endParam) error(400, 'No end date provided');

		const start = new Date(startParam);
		const end = new Date(endParam);
		if (!Number.isFinite(start.getTime()) || !Number.isFinite(end.getTime()) || end <= start) {
			error(400, 'Invalid calendar date range');
		}

		if (!calendar) {
			const endpoint = dependencies.getCalendarEndpoint();
			if (!endpoint) {
				error(500, 'No calendar endpoint configured.');
			}
			calendar = dependencies.createCalendar(endpoint, fetch);
		}

		let data;
		try {
			data = await calendar.getEventsBetween(start, end);
		} catch {
			calendar = undefined;
			// Fetch errors can contain the private ICS URL, so do not log the caught value.
			console.error('Failed to read staff calendar data.');
			error(500, 'Internal Service Error');
		}

		return new Response(JSON.stringify(data), {
			headers: {
				'Content-Type': 'application/json',
				'Cache-Control': 'private, no-store'
			}
		});
	};
}

const staffCalendarHandler = _createStaffCalendarHandler({
	getCalendarEndpoint: () => privateEnv.STAFF_CALENDAR_ENDPOINT,
	getLeashEndpoint: () => publicEnv.PUBLIC_LEASH_ENDPOINT,
	authorize: requireStaffCalendarAccess,
	createCalendar: (endpoint) => {
		const snapshot = getStaffCalendarSnapshot(endpoint);
		return {
			getEventsBetween: async (start, end) =>
				(await snapshot.getEventsBetween(start, end)).map(colorizeStaffCalendarEvent)
		};
	}
});

export const GET: RequestHandler = async ({ fetch, url, cookies }) =>
	staffCalendarHandler({ fetch, url, cookies });
