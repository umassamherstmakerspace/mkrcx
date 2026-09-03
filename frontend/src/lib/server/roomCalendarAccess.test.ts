import { describe, expect, it, vi } from 'vitest';
import { requireRoomCalendarAccess } from '$lib/server/roomCalendarAccess';

function leashUser(role: string) {
	return {
		ID: 7,
		CreatedAt: '2026-01-01T00:00:00Z',
		UpdatedAt: '2026-01-01T00:00:00Z',
		Email: 'person@example.edu',
		CardID: '',
		Name: 'Person',
		Pronouns: '',
		Role: role,
		Type: 'undergrad',
		GraduationYear: 2028,
		Major: '',
		Department: '',
		JobTitle: '',
		Permissions: []
	};
}

describe('room calendar server authorization', () => {
	it('rejects an unauthenticated request without calling Leash', async () => {
		const fetch = vi.fn();
		await expect(
			requireRoomCalendarAccess({ token: undefined, leashURL: 'https://leash.example', fetch })
		).rejects.toMatchObject({ status: 401 });
		expect(fetch).not.toHaveBeenCalled();
	});

	it.each(['member', 'volunteer', 'staff', 'admin'])(
		'allows an authenticated %s account',
		async (role) => {
			let requestInit: RequestInit | undefined;
			const fetch = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
				requestInit = init;
				return Response.json(leashUser(role));
			});
			const user = await requireRoomCalendarAccess({
				token: `${role}-token`,
				leashURL: 'https://leash.example',
				fetch: fetch as typeof globalThis.fetch
			});
			expect(user.role).toBe(role);
			expect(new Headers(requestInit?.headers).get('authorization')).toBe(`Bearer ${role}-token`);
		}
	);

	it('treats an invalid or expired token as unauthenticated', async () => {
		const fetch = vi.fn(async () => new Response('invalid token', { status: 401 }));
		await expect(
			requireRoomCalendarAccess({
				token: 'expired-token',
				leashURL: 'https://leash.example',
				fetch: fetch as typeof globalThis.fetch
			})
		).rejects.toMatchObject({ status: 401 });
	});
});
