import { LeashAPI } from '$lib/leash';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ data, fetch }) => {
	const { token, leashURL } = data;

	const api = new LeashAPI(token || '', leashURL);
    api.overrideFetchFunction(fetch);

	const user =
		token === undefined ? null : await api.selfUser({ withNotifications: true, withHolds: true });

	return {
		api,
		user
	};
};
