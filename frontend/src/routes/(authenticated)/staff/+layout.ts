import type { LayoutLoad } from './$types';
import { error } from '@sveltejs/kit';

export const load: LayoutLoad = async ({ parent }) => {
	const { user } = await parent();

	if (!user.isStaff) {
		error(403, 'You do not have permission to access this page.');
	}

	return { user };
};
