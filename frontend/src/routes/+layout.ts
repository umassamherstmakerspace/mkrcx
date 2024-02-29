import type { LayoutLoad } from './$types';
import { LeashAPI } from '$lib/leash';
import { env } from '$env/dynamic/public';
import Cookies from 'js-cookie';

export const ssr = false;

export const load: LayoutLoad = async () => {
	const token = Cookies.get('token') || '';
	const leashURL = env.PUBLIC_LEASH_ENDPOINT;
	if (!leashURL) {
		throw new Error('LEASH_ENDPOINT not set');
	}

	const api = new LeashAPI(token, leashURL);
	let u;

	try {
		u = await api.selfUser({ withNotifications: true, withHolds: true });
	} catch (e) {
		u = null;
	}

	return {
		api,
		user: u
	};
};
