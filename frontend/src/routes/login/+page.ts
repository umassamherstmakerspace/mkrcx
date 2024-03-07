import { base } from '$app/paths';
import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';
import Cookies from 'js-cookie';

export const load: PageLoad = async ({ parent, url }) => {
	const { user, api } = await parent();

	const root = url.origin + base;
	let previousPage = url.searchParams.get('return_to') || root;
	if (previousPage.includes('/login')) {
		previousPage = root;
	}

	const token = url.searchParams.get('token');
	const state = url.searchParams.get('state');
	const expires_at = url.searchParams.get('expires_at');

	if (token && state && expires_at) {
		Cookies.set('token', token, {
			expires: new Date(expires_at),
			sameSite: 'strict'
		});

		let ret = atob(state);

		if (ret.includes('/login')) {
			ret = root;
		}

		redirect(307, ret);
	} else {
		if (user) {
            redirect(307, previousPage);
		} else {
			redirect(307, api.login(url.origin + url.pathname, previousPage));
		}
	}
};
