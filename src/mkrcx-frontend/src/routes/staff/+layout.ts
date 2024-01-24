import type { LayoutLoad } from './$types';
import { redirect } from '@sveltejs/kit';
import { error } from '@sveltejs/kit';

export const load: LayoutLoad = async ({ parent }) => {
	const { user } = await parent(); 

    if (!user) {
        redirect(307, '/login');
    }

    if (!user.isStaff) {
        error(403, 'You do not have permission to access this page.');
    }

    return {user};
};