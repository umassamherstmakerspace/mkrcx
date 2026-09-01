import { error } from '@sveltejs/kit';
import { LeashAPI, LeashAPIError, type User } from '$lib/leash';

export interface StaffCalendarAccessOptions {
	token: string | undefined;
	leashURL: string | undefined;
	fetch: typeof globalThis.fetch;
}

export async function requireStaffCalendarAccess({
	token,
	leashURL,
	fetch
}: StaffCalendarAccessOptions): Promise<User> {
	if (!token) {
		error(401, 'Authentication required.');
	}
	if (!leashURL) {
		error(500, 'LEASH_ENDPOINT not set');
	}

	const api = new LeashAPI(token, leashURL);
	api.overrideFetchFunction(fetch);

	let user: User;
	try {
		// Verify the user and role with Leash on the server. Never trust a role
		// supplied by the browser when protecting this standalone JSON endpoint.
		user = await api.selfUser({}, true);
	} catch (caught) {
		if (caught instanceof LeashAPIError && caught.status === 401) {
			error(401, 'Authentication required.');
		}
		if (caught instanceof LeashAPIError && caught.status === 403) {
			error(403, 'You do not have permission to access this resource.');
		}
		error(502, 'Error communicating with Leash.');
	}

	if (!user.isStaff) {
		error(403, 'You do not have permission to access this resource.');
	}

	return user;
}
