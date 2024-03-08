import { base } from '$app/paths';
import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent, url }) => {
	const { user, api } = await parent();

	const root = url.origin + base;

	if (user) {
		redirect(307, api.logout(root));
	} else {
		redirect(307, root);
	}
};
