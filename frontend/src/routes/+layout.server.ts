import type { LayoutServerLoad } from './$types';
import { LeashAPI } from '$lib/leash';
import { env } from '$env/dynamic/public';
import { error } from '@sveltejs/kit';

export const load: LayoutServerLoad = async ({ fetch, cookies }) => {
	const token = cookies.get('token') || '';
	const leashURL = env.PUBLIC_LEASH_ENDPOINT;
	if (!leashURL) {
		throw new Error('LEASH_ENDPOINT not set');
	}

	const api = new LeashAPI(token, leashURL);
	api.overrideFetchFunction(fetch);

	if (token) {
		try {
			if (await api.validateToken()) {
				const refresh = await api.refreshTokens();
				cookies.set('token', refresh.token, {
					expires: new Date(refresh.expires_at),
					path: '/'
				});
			} else {
				cookies.delete('token', { path: '/' });
			}
		} catch (e) {
			error(500, 'Error communicating with Leash');
		}
	}

	return {
		token: cookies.get('token'),
		leashURL
	};
};
