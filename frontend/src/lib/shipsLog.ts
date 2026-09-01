export const SHIPS_LOG_FORM_URL =
	'https://docs.google.com/forms/d/1727cnfdO5DWuIW3VqtpM6E3i9Usc1Xbs91KFIf3afmw/viewform';

export function shipsLogAccountChooserUrl(): string {
	const chooser = new URL('https://accounts.google.com/AccountChooser');
	chooser.searchParams.set('continue', SHIPS_LOG_FORM_URL);
	return chooser.toString();
}
