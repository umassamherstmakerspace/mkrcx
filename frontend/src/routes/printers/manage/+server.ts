import { env } from '$env/dynamic/public';
import type { RequestHandler } from './$types';

const proxy: RequestHandler = async ({ request, cookies, fetch, url }) => {
	const token = cookies.get('token');
	if (!token) return new Response('Sign in with a printer management account.', { status: 401 });
	if (request.method !== 'GET' && request.headers.get('origin') !== url.origin)
		return new Response('Invalid request origin', { status: 403 });
	const id = url.searchParams.get('id');
	if (request.method === 'PUT' && (!id || !/^[a-z0-9][a-z0-9-]{0,79}$/.test(id)))
		return new Response('Invalid printer identity', { status: 400 });
	try {
		const body = request.method === 'PUT' ? await request.text() : undefined;
		if (body && body.length > 16384) return new Response('Record too large', { status: 413 });
		const response = await fetch(
			`${env.PUBLIC_LEASH_ENDPOINT}/api/printer-fleet/records${request.method === 'PUT' ? `/${id}` : ''}`,
			{
				method: request.method,
				headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
				body,
				signal: AbortSignal.timeout(5000)
			}
		);
		return new Response(await response.text(), {
			status: response.status,
			headers: {
				'Content-Type': response.headers.get('Content-Type') ?? 'text/plain',
				'Cache-Control': 'private, no-store'
			}
		});
	} catch {
		return new Response('Printer registry unavailable', { status: 503 });
	}
};
export const GET = proxy;
export const PUT = proxy;
