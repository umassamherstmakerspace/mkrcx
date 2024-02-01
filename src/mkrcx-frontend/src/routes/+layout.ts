import type { LayoutLoad } from './$types';
import { User } from '$lib/leash';

export const ssr = false;

export const load: LayoutLoad = async () => {
	let u;

	try {
		u = await User.self({ withNotifications: true, withHolds: true });
	} catch (e) {
		u = null;
	}

	return {
		user: u
	};
};
