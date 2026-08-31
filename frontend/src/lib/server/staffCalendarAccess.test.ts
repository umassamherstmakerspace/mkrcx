import { describe, expect, it, vi } from 'vitest';
import { requireStaffCalendarAccess } from '$lib/server/staffCalendarAccess';

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

describe('staff calendar server authorization', () => {
	it('rejects an unauthenticated request without calling Leash', async () => {
		const fetch = vi.fn();
		await expect(
			requireStaffCalendarAccess({ token: undefined, leashURL: 'https://leash.example', fetch })
		).rejects.toMatchObject({ status: 401 });
		expect(fetch).not.toHaveBeenCalled();
	});

	it('rejects an authenticated ordinary member', async () => {
		let requestInit: RequestInit | undefined;
		const fetch = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
			requestInit = init;
			return Response.json(leashUser('member'));
		});
		await expect(
			requireStaffCalendarAccess({
				token: 'member-token',
				leashURL: 'https://leash.example',
				fetch: fetch as typeof globalThis.fetch
			})
		).rejects.toMatchObject({ status: 403 });
		expect(new Headers(requestInit?.headers).get('authorization')).toBe('Bearer member-token');
	});

	it.each(['volunteer', 'staff', 'admin'])('allows an authenticated %s account', async (role) => {
		const fetch = vi.fn(async () => Response.json(leashUser(role)));
		const user = await requireStaffCalendarAccess({
			token: `${role}-token`,
			leashURL: 'https://leash.example',
			fetch: fetch as typeof globalThis.fetch
		});
		expect(user.role).toBe(role);
		expect(user.isStaff).toBe(true);
	});

	it('treats an invalid or expired token as unauthenticated', async () => {
		const fetch = vi.fn(async () => new Response('invalid token', { status: 401 }));
		await expect(
			requireStaffCalendarAccess({
				token: 'expired-token',
				leashURL: 'https://leash.example',
				fetch: fetch as typeof globalThis.fetch
			})
		).rejects.toMatchObject({ status: 401 });
	});
});
