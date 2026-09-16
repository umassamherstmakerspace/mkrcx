import type { PageServerLoad } from './$types';
import { readPrinterDetail } from '$lib/server/printerDetail';

export const load: PageServerLoad = async ({ fetch, parent, params, setHeaders }) => {
	setHeaders({ 'Cache-Control': 'private, no-store' });
	const { token, leashURL } = await parent();
	return readPrinterDetail(fetch, leashURL, token, params.id);
};
