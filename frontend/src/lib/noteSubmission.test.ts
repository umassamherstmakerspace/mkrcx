import { describe, expect, it, vi } from 'vitest';
import { LeashAPI, LeashAPIError } from './leash';

describe('note submission', () => {
	it('sends text with authentication and an idempotency key', async () => {
		const request = vi.fn<[RequestInfo | URL, RequestInit?], Promise<Response>>();
		request.mockResolvedValue(new Response(null, { status: 201 }));
		const api = new LeashAPI('session-token', 'https://leash.example.test');
		api.overrideFetchFunction(request as typeof fetch);

		await api.submitNote('A useful note', '12f4fd94-4a8a-4d3c-9f9f-61e04274cfb5');
		expect(request).toHaveBeenCalledOnce();
		const [url, options] = request.mock.calls[0]!;
		expect(url).toBe('https://leash.example.test/api/notes');
		expect(options?.method).toBe('POST');
		expect(options?.headers).toMatchObject({
			Authorization: 'Bearer session-token',
			'Content-Type': 'application/json',
			'Idempotency-Key': '12f4fd94-4a8a-4d3c-9f9f-61e04274cfb5'
		});
		expect(options?.body).toBe(JSON.stringify({ note: 'A useful note' }));
	});

	it('preserves backend failure status for the form', async () => {
		const request = vi.fn<[RequestInfo | URL, RequestInit?], Promise<Response>>();
		request.mockResolvedValue(new Response('Please wait.', { status: 429 }));
		const api = new LeashAPI('session-token', 'https://leash.example.test');
		api.overrideFetchFunction(request as typeof fetch);

		await expect(api.submitNote('A useful note', 'attempt')).rejects.toEqual(
			new LeashAPIError(429, 'Please wait.')
		);
	});
});
