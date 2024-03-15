import { base } from '$app/paths';
import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { LeashAPI } from '$lib/leash';

export const load: PageServerLoad = async ({ parent, fetch, url, cookies }) => {
	const { token, leashURL } = await parent();

	const root = url.origin + base;

	if (token !== undefined) {
		const api = new LeashAPI(token, leashURL);
		api.overrideFetchFunction(fetch);

		cookies.delete('token', { path: '/' });
		redirect(307, api.logout(root));
	} else {
		redirect(307, root);
	}
};
