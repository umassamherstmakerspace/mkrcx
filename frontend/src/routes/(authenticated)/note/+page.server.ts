import { randomUUID } from 'node:crypto';
import { env } from '$env/dynamic/public';
import { fail, redirect } from '@sveltejs/kit';
import { LeashAPI, LeashAPIError } from '$lib/leash';
import type { Actions, PageServerLoad } from './$types';

const NOTE_MAXIMUM_CHARACTERS = 1500;

export const load: PageServerLoad = async () => ({ submissionId: randomUUID() });

export const actions: Actions = {
	default: async ({ request, cookies, fetch }) => {
		const form = await request.formData();
		const note = String(form.get('note') ?? '');
		const submissionId = String(form.get('submission_id') ?? '');
		const trimmed = note.trim();
		if (!trimmed) {
			return fail(400, { note, submissionId, message: 'Please enter a note.' });
		}
		if ([...trimmed].length > NOTE_MAXIMUM_CHARACTERS) {
			return fail(400, {
				note,
				submissionId,
				message: `Please keep your note to ${NOTE_MAXIMUM_CHARACTERS} characters or fewer.`
			});
		}
		if (
			!/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(submissionId)
		) {
			return fail(400, {
				note,
				submissionId: randomUUID(),
				message: 'Please try submitting again.'
			});
		}

		const token = cookies.get('token');
		if (!token) {
			redirect(303, '/login?return_to=%2Fnote');
		}
		const leashURL = env.PUBLIC_LEASH_ENDPOINT;
		if (!leashURL) {
			return fail(500, {
				note,
				submissionId,
				message: 'Note submission is temporarily unavailable.'
			});
		}
		const api = new LeashAPI(token, leashURL);
		api.overrideFetchFunction(fetch);
		try {
			await api.submitNote(trimmed, submissionId);
		} catch (error) {
			if (error instanceof LeashAPIError && error.status === 401) {
				redirect(303, '/login?return_to=%2Fnote');
			}
			if (error instanceof LeashAPIError && error.status === 429) {
				return fail(429, {
					note,
					submissionId,
					message: 'You have sent several notes recently. Please wait a few minutes and try again.'
				});
			}
			return fail(500, {
				note,
				submissionId,
				message: 'Your note was not saved. Please try again.'
			});
		}

		return { success: true, submissionId: randomUUID() };
	}
};
