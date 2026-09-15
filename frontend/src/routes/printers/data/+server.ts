import { env } from '$env/dynamic/public';
import { readFleet } from '$lib/server/printerFleet';
import type { RequestHandler } from './$types';
export const GET: RequestHandler = async ({ cookies, fetch }) => {
	try {
		if (!env.PUBLIC_LEASH_ENDPOINT) throw new Error('Missing registry endpoint');
		return Response.json(await readFleet(fetch, env.PUBLIC_LEASH_ENDPOINT, cookies.get('token')), {
			headers: { 'Cache-Control': 'private, no-store', 'X-Content-Type-Options': 'nosniff' }
		});
	} catch {
		return new Response('Printer registry unavailable', { status: 503 });
	}
};
