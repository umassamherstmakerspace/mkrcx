import { describe, expect, it } from 'vitest';
import { REGISTRATION_FORM_URL, registrationAccountChooserUrl } from '$lib/registration';

describe('Registration short URL', () => {
	it('uses the Google account chooser and preserves the exact form destination', () => {
		const redirect = new URL(registrationAccountChooserUrl());
		expect(redirect.origin).toBe('https://accounts.google.com');
		expect(redirect.pathname).toBe('/AccountChooser');
		expect(redirect.searchParams.get('continue')).toBe(REGISTRATION_FORM_URL);
	});
});
