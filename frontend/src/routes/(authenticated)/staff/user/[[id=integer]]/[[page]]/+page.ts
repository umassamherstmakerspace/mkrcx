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

	const { user, api } = await parent();

	let target: User;
	try {
		target = await api.userFromID(userID);
	} catch (e) {
		console.error(e);
		error(404, 'User not found.');
	}

	const tabs = {
		profile: true,
		trainings: true,
		holds: true,
		notifications: true,
		updates: true,
		apikeys: false,
		admin: false
	};

	if (user.role === 'admin') {
		tabs.apikeys = true;
		tabs.admin = true;
	}

	const tabsOpen = Object.fromEntries(Object.keys(tabs).map((tab) => [tab, false])) as Record<
		keyof typeof tabs,
		boolean
	>;
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

	return {
		target,
		tabs,
		tabsOpen
	};
};
