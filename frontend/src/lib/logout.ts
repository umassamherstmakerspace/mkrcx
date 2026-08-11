import { LeashAPIError, type LeashAPI } from './leash';

export async function revokeLoginSession(api: Pick<LeashAPI, 'logout'>): Promise<void> {
	try {
		await api.logout();
	} catch (error) {
		// A missing server-side session already satisfies logout. Other failures
		// must remain visible so the UI never claims revocation it could not confirm.
		if (!(error instanceof LeashAPIError) || error.status !== 401) {
			throw error;
		}
	}
}
