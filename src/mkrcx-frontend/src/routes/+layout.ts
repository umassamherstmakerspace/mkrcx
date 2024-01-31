import type { LayoutLoad } from './$types';
import { User } from '$lib/leash';
import Cookies from 'js-cookie';
import { derived, writable, type Readable, type Writable } from 'svelte/store';
import { setContext } from 'svelte';

type Themes = 'light' | 'dark' | 'system';
type DateTimeFormats = 'ISO' | 'US' | 'EU';

export const ssr = false;

export const load: LayoutLoad = async () => {
	const theme: Writable<Themes> = writable('system');
	const isDark: Readable<boolean> = derived(theme, ($theme) => {
		switch ($theme) {
			case 'light':
				return false;
			case 'dark':
				return true;
			case 'system':
				return window.matchMedia('(prefers-color-scheme: dark)').matches;
			default:
				return false;
		}
	});

	const dateLocale: Writable<DateTimeFormats> = writable('ISO');

	const dateFormat: Readable<string> = derived(dateLocale, ($dateLocale) => {
		switch ($dateLocale) {
			case 'US':
				return 'MM/dd/yyyy';
			case 'EU':
				return 'dd/MM/yyyy';
			case 'ISO':
			default:
				return 'yyyy-MM-dd';
		}
	});

	const timeFormat: Readable<string> = derived(dateLocale, ($dateLocale) => {
		switch ($dateLocale) {
			case 'US':
				return 'hh:mm:ss a';
			case 'EU':
				return 'HH:mm:ss';
			case 'ISO':
			default:
				return 'HH:mm:ssXXX';
		}
	});

	const dateTimeJoiner: Readable<string> = derived(dateLocale, ($dateLocale) => {
		switch ($dateLocale) {
			case 'US':
			case 'EU':
				return ' ';
			case 'ISO':
			default:
				return "'T'";
		}
	});

	const dateTimeFormat: Readable<string> = derived(
		[dateFormat, timeFormat, dateTimeJoiner],
		([$dateFormat, $timeFormat, $dateTimeJoiner]) => {
			return `${$dateFormat}${$dateTimeJoiner}${$timeFormat}`;
		}
	);

	theme.set((Cookies.get('theme') as Themes) || 'system');

	theme.subscribe((value) => {
		Cookies.set('theme', value, {
			expires: 365,
			sameSite: 'strict'
		});
	});

	dateLocale.set((Cookies.get('dateLocal') as DateTimeFormats) || 'ISO');

	dateLocale.subscribe((value) => {
		Cookies.set('dateLocal', value, {
			expires: 365,
			sameSite: 'strict'
		});
	});

	setContext('theme', theme);
	setContext('isDark', isDark);
	setContext('dateLocale', dateLocale);
	setContext('dateFormat', dateFormat);
	setContext('timeFormat', timeFormat);
	setContext('dateTimeJoiner', dateTimeJoiner);
	setContext('dateTimeFormat', dateTimeFormat);

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
