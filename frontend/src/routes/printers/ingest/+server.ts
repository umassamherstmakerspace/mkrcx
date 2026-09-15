import { env } from '$env/dynamic/private';
import { env as publicEnv } from '$env/dynamic/public';
import { tokenMatches } from '$lib/server/printerFleet';
import type { RequestHandler } from './$types';
export const POST: RequestHandler = async ({ request, fetch }) => {
	if (!tokenMatches(request.headers.get('authorization'), env.PRINTER_FLEET_INGEST_SECRET))
		return new Response('Unauthorized', { status: 401 });
	// Once the separate staging collector is active, acknowledge legacy uploads without
	// applying them. The shared legacy collector can keep serving production unchanged.
	if (env.PRINTER_REGISTRY_DEDICATED_COLLECTOR === 'true')
		return new Response(null, { status: 204 });
	try {
		const body = await request.text();
		if (body.length > 1_048_576) return new Response('Snapshot too large', { status: 413 });
		const response = await fetch(`${publicEnv.PUBLIC_LEASH_ENDPOINT}/api/printer-fleet/ingest`, {
			method: 'POST',
			headers: {
				Authorization: `Bearer ${env.PRINTER_FLEET_INGEST_SECRET}`,
				'Content-Type': 'application/json'
			},
			body,
			signal: AbortSignal.timeout(5000)
		});
		return new Response(null, { status: response.status });
	} catch {
		return new Response('Registry unavailable', { status: 503 });
	}
};
