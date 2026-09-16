import { json } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { readPrinterDetail } from '$lib/server/printerDetail';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ fetch, cookies, params, setHeaders }) => {
	setHeaders({ 'Cache-Control': 'private, no-store' });
	return json(
		await readPrinterDetail(fetch, env.PUBLIC_LEASH_ENDPOINT, cookies.get('token'), params.id)
	);
};
