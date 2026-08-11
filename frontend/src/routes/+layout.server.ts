import type { LayoutServerLoad } from './$types';
import { LeashAPI, LeashAPIError } from '$lib/leash';
import { env } from '$env/dynamic/public';
import { error } from '@sveltejs/kit';

export const load: LayoutServerLoad = async ({ fetch, cookies, url }) => {
	let token = cookies.get('token');
	const leashURL = env.PUBLIC_LEASH_ENDPOINT;
	if (!leashURL) {
		throw new Error('LEASH_ENDPOINT not set');
	}

	const api = new LeashAPI(token || '', leashURL);
	api.overrideFetchFunction(fetch);

	if (token) {
		try {
			if (await api.validateToken()) {
				const refresh = await api.refreshTokens();
				token = refresh.token;
				cookies.set('token', refresh.token, {
					expires: new Date(refresh.expires_at),
					httpOnly: true,
					secure: url.protocol === 'https:',
					sameSite: 'lax',
					path: '/'
				});
			} else {
				cookies.delete('token', { path: '/' });
				token = undefined;
			}
		} catch (e) {
			if (e instanceof LeashAPIError && e.status === 401) {
				cookies.delete('token', { path: '/' });
				token = undefined;
			} else {
				error(500, 'Error communicating with Leash');
			}
		}
	}

	return {
		token,
		leashURL
	};
};
