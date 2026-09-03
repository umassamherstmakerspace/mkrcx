import { env } from '$env/dynamic/private';
import { acceptSnapshot, tokenMatches } from '$lib/server/printerFleet';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ request }) => {
	if (!tokenMatches(request.headers.get('authorization'), env.PRINTER_FLEET_INGEST_SECRET))
		return new Response('Unauthorized', { status: 401 });
	try {
		acceptSnapshot(await request.json());
		return new Response(null, { status: 204 });
	} catch {
		return new Response('Invalid snapshot', { status: 400 });
	}
};
