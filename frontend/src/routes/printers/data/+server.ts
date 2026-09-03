import { env } from '$env/dynamic/public';
import { fleetResponse } from '$lib/server/printerFleet';
import { requireStaffCalendarAccess } from '$lib/server/staffCalendarAccess';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ cookies, fetch }) => {
	let staff = false;
	const token = cookies.get('token');
	if (token) {
		try {
			await requireStaffCalendarAccess({ token, leashURL: env.PUBLIC_LEASH_ENDPOINT, fetch });
			staff = true;
		} catch {
			staff = false;
		}
	}
	return Response.json(fleetResponse(staff), {
		headers: { 'Cache-Control': 'private, no-store', 'X-Content-Type-Options': 'nosniff' }
	});
};
