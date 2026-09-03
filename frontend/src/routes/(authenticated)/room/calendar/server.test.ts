import { error } from '@sveltejs/kit';
import { describe, expect, it, vi } from 'vitest';
import { _createRoomCalendarHandler } from './+server';

const secretEndpoint = 'https://calendar.example/private-room-credential.ics';

function context(token: string | undefined) {
	return {
		fetch: vi.fn() as unknown as typeof globalThis.fetch,
		url: new URL('https://mkr.cx/room/calendar?start=2026-09-08&end=2026-09-09'),
		cookies: { get: () => token }
	};
}

function handlerForRole(role: 'anonymous' | 'member' | 'staff') {
	const createCalendar = vi.fn(() => ({
		getEventsBetween: vi.fn(async () => [
			{
				title: 'Private project review',
				description: 'Confidential booking notes',
				location: 'Ag Eng 119',
				start: '2026-09-08T09:00:00-04:00',
				end: '2026-09-08T10:00:00-04:00',
				allDay: false,
				uid: 'private-calendar-uid'
			}
		])
	}));
	const handler = _createRoomCalendarHandler({
		getCalendarEndpoint: () => secretEndpoint,
		getLeashEndpoint: () => 'https://leash.example',
		authorize: async () => {
			if (role === 'anonymous') error(401, 'Authentication required.');
			return { isStaff: role === 'staff' };
		},
		createCalendar
	});
	return { handler, createCalendar };
}

describe('/room/calendar endpoint', () => {
	it('rejects a logged-out request before reading the private calendar', async () => {
		const { handler, createCalendar } = handlerForRole('anonymous');
		await expect(handler(context(undefined))).rejects.toMatchObject({ status: 401 });
		expect(createCalendar).not.toHaveBeenCalled();
	});

	it('returns only free/busy fields to an authenticated ordinary member', async () => {
		const { handler } = handlerForRole('member');
		const response = await handler(context('member-token'));
		const body = await response.text();

		expect(response.status).toBe(200);
		expect(response.headers.get('cache-control')).toBe('private, no-store');
		expect(JSON.parse(body)).toEqual([
			{
				title: 'Busy',
				start: '2026-09-08T09:00:00-04:00',
				end: '2026-09-08T10:00:00-04:00',
				allDay: false,
				backgroundColor: '#64748b',
				borderColor: '#64748b'
			}
		]);
		expect(body).not.toContain('Private project review');
		expect(body).not.toContain('Confidential booking notes');
		expect(body).not.toContain('private-calendar-uid');
		expect(body).not.toContain(secretEndpoint);
	});

	it('returns booking details to an authenticated staff account', async () => {
		const { handler } = handlerForRole('staff');
		const response = await handler(context('staff-token'));
		const body = await response.text();

		expect(JSON.parse(body)).toEqual([
			expect.objectContaining({
				title: 'Private project review',
				description: 'Confidential booking notes',
				location: 'Ag Eng 119',
				uid: 'private-calendar-uid'
			})
		]);
		expect(body).not.toContain(secretEndpoint);
	});
});
