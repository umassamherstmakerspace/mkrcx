import { env } from '$env/dynamic/public';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ cookies, fetch, url }) => {
	const token = cookies.get('token');
	const id = url.searchParams.get('id');
	const headers = { 'Cache-Control': 'private, no-store' };
	if (!token) return new Response('Sign in to view printer history.', { status: 401, headers });
	if (!id || !/^[a-z0-9][a-z0-9-]{0,79}$/.test(id))
		return new Response('Invalid printer', { status: 400, headers });
	try {
		const response = await fetch(`${env.PUBLIC_LEASH_ENDPOINT}/api/printer-fleet/history/${id}`, {
			headers: { Authorization: `Bearer ${token}` },
			signal: AbortSignal.timeout(5000)
		});
		return new Response(await response.text(), {
			status: response.status,
			headers: { ...headers, 'Content-Type': response.headers.get('Content-Type') ?? 'text/plain' }
		});
	} catch {
		return new Response('Printer history is temporarily unavailable.', { status: 503, headers });
	}
};
