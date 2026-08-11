import { error } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent }) => {
	const { user } = await parent();

	if (!user.canExportCheckins) {
		error(403, 'You do not have permission to export check-in data.');
	}

	return { user };
};
