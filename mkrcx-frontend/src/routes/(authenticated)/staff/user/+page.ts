import type { PageLoad } from './$types';
import { error } from '@sveltejs/kit';

export const load: PageLoad = async () => {
	error(400, 'No user ID provided.');
};
