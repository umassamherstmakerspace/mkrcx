import type { PageLoad } from './$types';
import { User } from '$lib/leash';
import { theme, dateTimeFormat, user, type Themes, ISODate } from '$lib/stores';
import Cookies from 'js-cookie';

export const ssr = false;

export const load: PageLoad = async () => {
	theme.set((Cookies.get('theme') as Themes) || 'system');

	theme.subscribe((value) => {
		Cookies.set('theme', value,
		{
			expires: 365,
			sameSite: 'strict'
		});
	});

	dateTimeFormat.set((Cookies.get('dateTimeFormat') as string) || ISODate);

	dateTimeFormat.subscribe((value) => {
		Cookies.set('dateTimeFormat', value,
		{
			expires: 365,
			sameSite: 'strict'
		});
	});

	let u;

	try {
		u = await User.self();
	} catch (e) {
		u = null;
	}

	user.set(u);

	return {
		user: u
	};
};
