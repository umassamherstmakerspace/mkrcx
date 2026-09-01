import type { ServerInit } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import { initializeStaffCalendarSnapshot } from '$lib/server/staffCalendarSnapshot';

export const init: ServerInit = async () => {
	if (!env.STAFF_CALENDAR_ENDPOINT) return;

	try {
		await initializeStaffCalendarSnapshot(env.STAFF_CALENDAR_ENDPOINT);
	} catch {
		// Keep the site available if the source is temporarily unavailable. The
		// first staff-calendar request will retry without exposing the private URL.
		console.error('Failed to prewarm staff calendar snapshot.');
	}
};
