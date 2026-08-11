import { base } from '$app/paths';
import { env } from '$env/dynamic/public';
import { LeashAPI } from '$lib/leash';
import { revokeLoginSession } from '$lib/logout';
import { error, redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ cookies, fetch, setHeaders, url }) => {
	setHeaders({
		'cache-control': 'no-store',
		'referrer-policy': 'no-referrer'
	});

	const leashURL = env.PUBLIC_LEASH_ENDPOINT;
	if (!leashURL) {
		error(500, 'LEASH_ENDPOINT not set');
	}
	const token = cookies.get('token');
	if (token) {
		const api = new LeashAPI(token, leashURL);
		api.overrideFetchFunction(fetch);
		try {
			await revokeLoginSession(api);
		} catch {
			error(502, 'Logout could not be confirmed by Leash. Please try again.');
		}
	}

	cookies.delete('token', { path: '/' });
	redirect(303, url.origin + base);
};
