import { describe, expect, it } from 'vitest';
import { SHIPS_LOG_FORM_URL, shipsLogAccountChooserUrl } from '$lib/shipsLog';

describe("Ship's Log short URL", () => {
	it('uses the Google account chooser and preserves the exact form destination', () => {
		const redirect = new URL(shipsLogAccountChooserUrl());
		expect(redirect.origin).toBe('https://accounts.google.com');
		expect(redirect.pathname).toBe('/AccountChooser');
		expect(redirect.searchParams.get('continue')).toBe(SHIPS_LOG_FORM_URL);
	});
});
