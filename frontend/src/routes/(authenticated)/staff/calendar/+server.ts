import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';
import { Cached } from '$lib/types';

let calCache: Cached<string>;

async function fetchCalendar(fetch: typeof globalThis.fetch) {
	const cal = env.STAFF_CALENDAR_ENDPOINT;
	if (!cal) {
		error(500, 'No calendar endpoint configured.');
	}

	const req = await fetch(cal);
	if (!req.ok) {
		error(500, 'Failed to fetch calendar.');
	}

	return await req.text();
}

export const GET: RequestHandler = async ({ fetch }) => {
	if (!calCache) {
		calCache = new Cached(() => fetchCalendar(fetch), 1000 * 60 * 5); // 5 minutes
	}

	const data = await calCache.get();
	return new Response(data, {
		headers: {
			'Content-Type': 'text/calendar'
		}
	});
};
