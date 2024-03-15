import type { LayoutLoad } from './$types';
import { redirect } from '@sveltejs/kit';

export const load: LayoutLoad = async ({ parent, url }) => {
	const { user } = await parent();

	if (!user) {
		redirect(307, '/login?return_to=' + encodeURIComponent(url.pathname));
	}

	return { user };
};
