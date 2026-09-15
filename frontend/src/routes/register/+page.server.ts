import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ parent, setHeaders }) => {
	setHeaders({ 'cache-control': 'no-store' });
	const { token } = await parent();

	if (token) redirect(303, '/');
};
