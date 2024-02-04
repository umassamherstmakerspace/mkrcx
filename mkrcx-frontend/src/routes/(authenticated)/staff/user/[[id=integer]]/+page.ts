import { User } from '$lib/leash';
import type { PageLoad } from './$types';
import { error } from '@sveltejs/kit';

export const load: PageLoad = async ({ parent, params }) => {
	if (!params.id) {
		error(400, 'No user ID provided.');
	}

	const userID = Number.parseInt(params.id);
	if (Number.isNaN(userID)) {
		error(400, 'Invalid user ID provided.');
	}

	try {
		const { user } = await parent();
		const target = await User.fromID(userID);

		const tabs = {
			profile: true,
			trainings: true,
			holds: true,
			updates: true,
			apikeys: false
		};

		if (user.role === 'staff') {
			tabs.apikeys = true;
		}

		const tabsOpen = Object.fromEntries(Object.entries(tabs)) as typeof tabs;

		return {
			target,
			tabs,
			tabsOpen
		};
	} catch (e) {
		console.error(e);
		error(404, 'User not found.');
	}
};
