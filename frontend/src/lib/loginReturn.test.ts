import { describe, expect, it } from 'vitest';
import { LeashAPI, LeashAPIError } from './leash';
import { safeLoginReturn } from './loginReturn';
import { revokeLoginSession } from './logout';

const origin = 'https://staging.mkr.cx';
const fallback = 'https://staging.mkr.cx/';

describe('safeLoginReturn', () => {
	it('allows relative and same-origin destinations', () => {
		expect(safeLoginReturn('/staff/feeds/1', origin, fallback)).toBe(
			'https://staging.mkr.cx/staff/feeds/1'
		);
		expect(safeLoginReturn('https://staging.mkr.cx/profile?tab=holds', origin, fallback)).toBe(
			'https://staging.mkr.cx/profile?tab=holds'
		);
	});

	it.each([
		'https://attacker.example/collect',
		'//attacker.example/collect',
		'javascript:alert(1)',
		'https://user:password@staging.mkr.cx/'
	])('replaces unsafe destination %s with the fallback', (candidate) => {
		expect(safeLoginReturn(candidate, origin, fallback)).toBe(fallback);
	});
});

describe('login code exchange', () => {
	it('redeems without sending an empty bearer header', async () => {
		const api = new LeashAPI('', 'https://leash.staging.mkr.cx');
		let request: RequestInit | undefined;
		api.overrideFetchFunction((async (_input, init) => {
			request = init;
			return new Response(
				JSON.stringify({ token: 'session-token', expires_at: '2026-08-12T12:00:00Z' }),
				{ status: 200, headers: { 'Content-Type': 'application/json' } }
			);
		}) as typeof fetch);

		await expect(api.exchangeLoginCode('one-time-code')).resolves.toEqual({
			token: 'session-token',
			expires_at: '2026-08-12T12:00:00Z'
		});
		expect(new Headers(request?.headers).has('Authorization')).toBe(false);
		expect(request?.body).toBe(JSON.stringify({ code: 'one-time-code' }));
	});
});

describe('logout', () => {
	it('uses an authenticated POST without putting the bearer token in the URL', async () => {
		const api = new LeashAPI('session-token', 'https://leash.staging.mkr.cx');
		let requestURL = '';
		let request: RequestInit | undefined;
		api.overrideFetchFunction((async (input, init) => {
			requestURL = String(input);
			request = init;
			return new Response(null, { status: 204 });
		}) as typeof fetch);

		await api.logout();

		expect(requestURL).toBe('https://leash.staging.mkr.cx/auth/logout');
		expect(requestURL).not.toContain('session-token');
		expect(request?.method).toBe('POST');
		expect(new Headers(request?.headers).get('Authorization')).toBe('Bearer session-token');
	});

	it('treats an already-missing server session as successfully revoked', async () => {
		await expect(
			revokeLoginSession({
				logout: async () => {
					throw new LeashAPIError(401, 'session already expired');
				}
			})
		).resolves.toBeUndefined();
	});

	it('does not claim logout when server-side revocation fails', async () => {
		await expect(
			revokeLoginSession({
				logout: async () => {
					throw new LeashAPIError(503, 'temporarily unavailable');
				}
			})
		).rejects.toEqual(expect.objectContaining<Partial<LeashAPIError>>({ status: 503 }));
	});
});
