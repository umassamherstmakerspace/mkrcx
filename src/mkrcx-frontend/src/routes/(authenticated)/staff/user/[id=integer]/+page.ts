import { User } from '$lib/leash';
import type { PageLoad } from './$types';
import { error } from '@sveltejs/kit';

export const load: PageLoad = async ({ params }) => {
    if (!params.id) {
        error(400, 'No user ID provided.');
    }

    const userID = Number.parseInt(params.id);
    if (Number.isNaN(userID)) {
        error(400, 'Invalid user ID provided.');
    }

    try {
        const target = await User.fromID(userID);
        if (target.role === 'sevice') {
            error(403, 'You cannot view service accounts with this page.');
        }
        
        return {
            target
        };
    } catch (e) {
        error(404, 'User not found.');
    }
};