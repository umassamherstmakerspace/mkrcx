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

	let target: User;
	try {
		target = await User.fromID(userID);
	} catch (e) {
		console.error(e);
		error(404, 'User not found.');
	}

	const { user } = await parent();

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

	const tabsOpen = Object.fromEntries(Object.keys(tabs).map((tab) => [tab, false])) as Record<keyof typeof tabs, boolean>;
	type tabsType = keyof typeof tabs;

	if (params.page) {
		const page = params.page as tabsType;
		if (tabs[page]) {
			tabsOpen[page] = true;
		} else {
			error(404, 'Invalid page.');
		}
	} else {
		const firstTab = (Object.keys(tabs) as tabsType[]).find((tab) => tabs[tab]);
		if (firstTab) {
			tabsOpen[firstTab] = true;
		}
	}

	console.log('tabsOpen', tabsOpen);

	return {
		target,
		tabs,
		tabsOpen
	};
};
