import { error } from '@sveltejs/kit';
import { describe, expect, it, vi } from 'vitest';
import { _createStaffCalendarHandler } from './+server';

const secretEndpoint = 'https://calendar.example/private-secret.ics';

function context(token: string | undefined) {
	return {
		fetch: vi.fn() as unknown as typeof globalThis.fetch,
		url: new URL('https://mkr.cx/staff/calendar?start=2026-09-08&end=2026-09-09'),
		cookies: { get: () => token }
	};
}

function handlerForRole(role: 'anonymous' | 'member' | 'staff') {
	const createCalendar = vi.fn(() => ({
		getEventsBetween: vi.fn(async () => [
			{
				title: 'Niall',
				start: '2026-09-08T09:00:00-04:00',
				end: '2026-09-08T17:00:00-04:00',
				backgroundColor: '#cd74e6'
			}
		])
	}));
	const handler = _createStaffCalendarHandler({
		getCalendarEndpoint: () => secretEndpoint,
		getLeashEndpoint: () => 'https://leash.example',
		authorize: async () => {
			if (role === 'anonymous') error(401, 'Authentication required.');
			if (role === 'member') error(403, 'Forbidden.');
		},
		createCalendar
	});
	return { handler, createCalendar };
}

describe('/staff/calendar endpoint', () => {
	it('rejects a logged-out request before reading the private calendar', async () => {
		const { handler, createCalendar } = handlerForRole('anonymous');
		await expect(handler(context(undefined))).rejects.toMatchObject({ status: 401 });
		expect(createCalendar).not.toHaveBeenCalled();
	});

	it('rejects an authenticated non-staff member before reading the private calendar', async () => {
		const { handler, createCalendar } = handlerForRole('member');
		await expect(handler(context('member-token'))).rejects.toMatchObject({ status: 403 });
		expect(createCalendar).not.toHaveBeenCalled();
	});

	it('returns private, read-only JSON to an authenticated staff account without exposing the source URL', async () => {
		const { handler, createCalendar } = handlerForRole('staff');
		const response = await handler(context('staff-token'));
		const body = await response.text();

		expect(response.status).toBe(200);
		expect(response.headers.get('content-type')).toContain('application/json');
		expect(response.headers.get('cache-control')).toBe('private, no-store');
		expect(JSON.parse(body)).toEqual([
			expect.objectContaining({ title: 'Niall', backgroundColor: '#cd74e6' })
		]);
		expect(body).not.toContain(secretEndpoint);
		expect(createCalendar).toHaveBeenCalledWith(secretEndpoint, expect.any(Function));
	});
});
