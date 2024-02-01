<script lang="ts">
	import Header from '$lib/components/Header.svelte';
	import SideMenu from '$lib/components/SideMenu.svelte';
	import '../app.pcss';
	import type { LayoutData } from './$types';
	import Cookies from 'js-cookie';
	import { derived, writable, type Readable, type Writable } from 'svelte/store';
	import { setContext } from 'svelte';

	type Themes = 'light' | 'dark' | 'system';
	type DateTimeFormats = 'ISO' | 'US' | 'EU';

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

	isDark.subscribe((value) => {
		if (value) {
			document.documentElement.classList.add('dark');
		} else {
			document.documentElement.classList.remove('dark');
		}
	});

	let hideSidebar = true;

	export let data: LayoutData;
	let { user } = data;
</script>

<svelte:head>
	<link rel="icon" type="image/svg" href="/favicon.svg" />
</svelte:head>

<SideMenu bind:hidden={hideSidebar} {user} />
<div class="flex h-dvh w-dvw flex-col">
	<Header bind:hideSidebar {user} />
	<div class="h-full max-h-full w-full flex-1 overflow-auto p-4">
		<slot />
	</div>
</div>
