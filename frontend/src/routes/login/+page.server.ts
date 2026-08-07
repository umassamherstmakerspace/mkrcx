import { base } from '$app/paths';
import { error, redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { LeashAPI } from '$lib/leash';
import { safeLoginReturn } from '$lib/loginReturn';

export const load: PageServerLoad = async ({ parent, fetch, url, cookies, setHeaders }) => {
	const { token, leashURL } = await parent();
	setHeaders({
		'cache-control': 'no-store',
		'referrer-policy': 'no-referrer'
	});

	const root = url.origin + base;
	let previousPage = safeLoginReturn(url.searchParams.get('return_to'), url.origin, root);
	if (previousPage.includes('/login')) {
		previousPage = root;
	}

	const loginCode = url.searchParams.get('code');
	const state = url.searchParams.get('state');

	if (loginCode && state) {
		const api = new LeashAPI('', leashURL);
		api.overrideFetchFunction(fetch);
		try {
			const exchange = await api.exchangeLoginCode(loginCode);
			cookies.set('token', exchange.token, {
				expires: new Date(exchange.expires_at),
				httpOnly: true,
				secure: url.protocol === 'https:',
				sameSite: 'lax',
				path: '/'
			});
		} catch {
			error(401, 'This login link is invalid or expired. Please start again.');
		}
		let decodedState: string;
		try {
			decodedState = atob(state);
		} catch {
			decodedState = root;
		}
		const destination = safeLoginReturn(decodedState, url.origin, root);
		redirect(303, destination.includes('/login') ? root : destination);
	} else {
		if (token === undefined) {
			const api = new LeashAPI('', leashURL);
			api.overrideFetchFunction(fetch);
			redirect(303, api.login(url.origin + url.pathname, previousPage));
		} else {
			redirect(303, previousPage);
		}
	}
};
