export const REGISTRATION_FORM_URL =
	'https://docs.google.com/forms/d/e/1FAIpQLSe_LxxnbIhcgh_CYXfAzMhDQDk1AR6WW2pbJEu3_zsT8x4-8w/viewform';

export function registrationAccountChooserUrl(): string {
	const chooser = new URL('https://accounts.google.com/AccountChooser');
	chooser.searchParams.set('continue', REGISTRATION_FORM_URL);
	return chooser.toString();
}
