import { base } from '$app/paths';
import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { LeashAPI } from '$lib/leash';

export const load: PageServerLoad = async ({ parent, fetch, url, cookies }) => {
	const { token, leashURL } = await parent();

	const root = url.origin + base;
	let previousPage = url.searchParams.get('return_to') || root;
	if (previousPage.includes('/login')) {
		previousPage = root;
	}

	const loginToken = url.searchParams.get('token');
	const state = url.searchParams.get('state');
	const expires_at = url.searchParams.get('expires_at');

	if (loginToken && state && expires_at) {
		cookies.set('token', loginToken, {
			expires: new Date(expires_at),
			path: '/'
		});

		let ret = atob(state);

		if (ret.includes('/login')) {
			ret = root;
		}

		redirect(307, ret);
	} else {
		if (token === undefined) {
			const api = new LeashAPI('', leashURL);
			api.overrideFetchFunction(fetch);
			redirect(307, api.login(url.origin + url.pathname, previousPage));
		} else {
			redirect(307, previousPage);
		}
	}
};
