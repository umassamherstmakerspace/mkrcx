import { describe, expect, it } from 'vitest';
import { LeashAPI, LeashAPIError } from '$lib/leash';

describe('Check-in CSV client', () => {
	it('sends an authenticated UTC range and preserves the server filename', async () => {
		const api = new LeashAPI('staff-token', 'https://leash.example');
		let requestURL = '';
		let authorization = '';
		api.overrideFetchFunction((async (input, init) => {
			requestURL = String(input);
			authorization = new Headers(init?.headers).get('authorization') || '';
			return new Response('event_id,member_uuid\n1,stable-id\n', {
				status: 200,
				headers: {
					'content-type': 'text/csv',
					'content-disposition': 'attachment; filename="checkins-202608.csv"'
				}
			});
		}) as typeof fetch);

		const result = await api.downloadCheckinCSV(
			new Date('2026-08-01T04:00:00.000Z'),
			new Date('2026-08-12T04:00:00.000Z')
		);

		const url = new URL(requestURL);
		expect(url.pathname).toBe('/api/checkins/export.csv');
		expect(url.searchParams.get('start')).toBe('2026-08-01T04:00:00.000Z');
		expect(url.searchParams.get('end')).toBe('2026-08-12T04:00:00.000Z');
		expect(authorization).toBe('Bearer staff-token');
		expect(result.filename).toBe('checkins-202608.csv');
		expect(await result.blob.text()).toContain('stable-id');
	});

	it('surfaces a revoked export permission as a typed error', async () => {
		const api = new LeashAPI('staff-token', 'https://leash.example');
		api.overrideFetchFunction(
			(async () => new Response('export permission required', { status: 403 })) as typeof fetch
		);

		await expect(
			api.downloadCheckinCSV(new Date('2026-08-01T00:00:00Z'), new Date('2026-08-02T00:00:00Z'))
		).rejects.toEqual(
			expect.objectContaining<Partial<LeashAPIError>>({
				name: 'LeashAPIError',
				status: 403,
				message: 'export permission required'
			})
		);
	});
});
