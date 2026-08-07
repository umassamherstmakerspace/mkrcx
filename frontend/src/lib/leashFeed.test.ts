import { describe, expect, it } from 'vitest';
import { LeashAPI, LeashAPIError } from '$lib/leash';

describe('Leash feed client errors', () => {
	it('preserves the HTTP status needed for HUD revocation handling', async () => {
		const api = new LeashAPI('expired-token', 'http://127.0.0.1:8000');
		api.overrideFetchFunction(
			(async () => new Response('feed access revoked', { status: 403 })) as typeof fetch
		);

		await expect(api.feedFromID(1, true)).rejects.toEqual(
			expect.objectContaining<Partial<LeashAPIError>>({
				name: 'LeashAPIError',
				status: 403,
				message: 'feed access revoked'
			})
		);
	});

	it('treats only 401 as an invalid token', async () => {
		const unauthorized = new LeashAPI('expired-token', 'http://127.0.0.1:8000');
		unauthorized.overrideFetchFunction(
			(async () => new Response('expired', { status: 401 })) as typeof fetch
		);
		await expect(unauthorized.validateToken()).resolves.toBe(false);

		const unavailable = new LeashAPI('valid-token', 'http://127.0.0.1:8000');
		unavailable.overrideFetchFunction(
			(async () => new Response('temporarily unavailable', { status: 503 })) as typeof fetch
		);
		await expect(unavailable.validateToken()).rejects.toEqual(
			expect.objectContaining<Partial<LeashAPIError>>({ status: 503 })
		);
	});
});
