<script lang="ts">
	import Header from '$lib/components/Header.svelte';
	import SideMenu from '$lib/components/SideMenu.svelte';
	import '../app.pcss';
	import type { LayoutData } from './$types';
	import { web_storage } from 'svelte-web-storage';
	import { derived, type Readable, type Writable } from 'svelte/store';
	import { setContext } from 'svelte';
	import { ModeWatcher } from 'mode-watcher';

	type DateTimeFormats = 'ISO' | 'US' | 'EU';

	const dateLocale: Writable<DateTimeFormats> = web_storage('date_local', 'ISO');

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

	setContext('dateLocale', dateLocale);
	setContext('dateFormat', dateFormat);
	setContext('timeFormat', timeFormat);
	setContext('dateTimeJoiner', dateTimeJoiner);
	setContext('dateTimeFormat', dateTimeFormat);

	let hideSidebar = true;

	export let data: LayoutData;
	const { user } = data;
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

<ModeWatcher />
