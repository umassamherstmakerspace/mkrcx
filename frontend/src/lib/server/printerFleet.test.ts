import { describe, expect, it, vi } from 'vitest';
import { readFleet, tokenMatches } from './printerFleet';
describe('printer registry proxy', () => {
	it('rejects absent and wrong collector credentials', () => {
		expect(tokenMatches(null, 'secret')).toBe(false);
		expect(tokenMatches('Bearer wrong', 'secret')).toBe(false);
		expect(tokenMatches('Bearer secret', 'secret')).toBe(true);
	});
	it('falls back to public data for an invalid session without forwarding credentials', async () => {
		const fetch = vi
			.fn()
			.mockResolvedValueOnce(new Response(null, { status: 401 }))
			.mockResolvedValueOnce(Response.json({ audience: 'public', printers: [] }));
		expect(await readFleet(fetch, 'https://leash.example', 'expired')).toEqual({
			audience: 'public',
			printers: []
		});
		expect(fetch.mock.calls[1][1].headers).toBeUndefined();
	});
	it('accepts a changing roster and retains durable notes in the response', async () => {
		const response = {
			audience: 'public',
			stale: true,
			printers: [{ id: 'new-printer', note: 'Awaiting thermistor', condition: 'out' }]
		};
		expect(
			await readFleet(vi.fn().mockResolvedValue(Response.json(response)), 'https://leash.example')
		).toEqual(response);
	});
	it('does not turn backend failure into an empty successful fleet', async () => {
		await expect(
			readFleet(
				vi.fn().mockResolvedValue(new Response(null, { status: 503 })),
				'https://leash.example'
			)
		).rejects.toThrow();
	});
});
