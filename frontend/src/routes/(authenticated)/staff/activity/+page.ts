import { error } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent }) => {
	const { user, api } = await parent();
	if (!user.canReadActivity) {
		error(403, 'You do not have permission to view activity reporting.');
	}

	return {
		user,
		activity: await api.getActivity('semester')
	};
};
