import { describe, expect, it, vi } from 'vitest';
import { readPrinterDetail } from './printerDetail';

const endpoint = 'https://leash.example';
const printer = {
	id: 'ada',
	name: 'Ada',
	condition: 'out',
	note: 'Awaiting fan',
	connected: false
};
const fleet = { audience: 'staff', stale: false, printers: [printer] };

describe('staff printer details', () => {
	it('requires a session before contacting the backend', async () => {
		const fetch = vi.fn();
		await expect(readPrinterDetail(fetch, endpoint, undefined, 'ada')).rejects.toMatchObject({
			status: 401
		});
		expect(fetch).not.toHaveBeenCalled();
	});
	it.each([401, 403])(
		'does not fall back to public data after an access denial (%s)',
		async (status) => {
			const fetch = vi.fn().mockResolvedValue(new Response(null, { status }));
			await expect(readPrinterDetail(fetch, endpoint, 'token', 'ada')).rejects.toMatchObject({
				status
			});
			expect(fetch).toHaveBeenCalledTimes(1);
		}
	);
	it('rejects unknown or malformed printer identities', async () => {
		const fetch = vi.fn().mockResolvedValue(Response.json(fleet));
		await expect(readPrinterDetail(fetch, endpoint, 'token', '../ada')).rejects.toMatchObject({
			status: 404
		});
		expect(fetch).not.toHaveBeenCalled();
		await expect(readPrinterDetail(fetch, endpoint, 'token', 'missing')).rejects.toMatchObject({
			status: 404
		});
	});
	it('retains offline notes and separately checks editing permission', async () => {
		const fetch = vi
			.fn()
			.mockResolvedValueOnce(Response.json(fleet))
			.mockResolvedValueOnce(Response.json(['leash.printers:manage']));
		expect(await readPrinterDetail(fetch, endpoint, 'token', 'ada')).toMatchObject({
			printer,
			canManage: true
		});
		expect(fetch.mock.calls[0][1].headers).toEqual({ Authorization: 'Bearer token' });
	});
	it('allows reading without editing permission and carries stale status', async () => {
		const fetch = vi
			.fn()
			.mockResolvedValueOnce(Response.json({ ...fleet, stale: true }))
			.mockResolvedValueOnce(new Response(null, { status: 403 }));
		expect(await readPrinterDetail(fetch, endpoint, 'token', 'ada')).toMatchObject({
			printer: { ...printer, stale: true },
			canManage: false
		});
	});
	it('rejects a public or unavailable response', async () => {
		await expect(
			readPrinterDetail(
				vi.fn().mockResolvedValue(Response.json({ ...fleet, audience: 'public' })),
				endpoint,
				'token',
				'ada'
			)
		).rejects.toMatchObject({ status: 503 });
		await expect(
			readPrinterDetail(vi.fn().mockRejectedValue(new Error('offline')), endpoint, 'token', 'ada')
		).rejects.toMatchObject({ status: 503 });
	});
});
