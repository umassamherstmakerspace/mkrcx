<script lang="ts">
	import Calendar from '$lib/components/Calendar.svelte';
	import { Heading, P } from 'flowbite-svelte';
	import type { PageData } from './$types';

	export let data: PageData;
</script>

<div class="flex flex-col items-center justify-center gap-12 text-center md:px-16">
	<Heading tag="h1" class="mb-4" customSize="text-4xl font-extrabold  md:text-5xl lg:text-6xl"
		>Welcome to the Staff Portal</Heading
	>
	<P class="mb-6 text-lg dark:text-gray-400 sm:px-16 lg:text-xl xl:px-48"
		>Access staff resources here.</P
	>
	{#if data.user.canListFeeds}
		<a
			href="/staff/feeds/1"
			class="group w-full max-w-2xl rounded-2xl border border-blue-200 bg-blue-50 p-6 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-blue-400 hover:shadow-md focus:outline-none focus:ring-4 focus:ring-blue-200 dark:border-blue-900 dark:bg-blue-950 dark:focus:ring-blue-900"
		>
			<span
				class="text-sm font-semibold uppercase tracking-[0.16em] text-blue-700 dark:text-blue-300"
				>Live staff tool</span
			>
			<span class="mt-2 flex items-center justify-between gap-4">
				<span>
					<span class="block text-2xl font-bold text-gray-950 dark:text-white"
						>Front Desk Check-in HUD</span
					>
					<span class="mt-1 block text-gray-700 dark:text-gray-200"
						>Monitor card taps and participation-agreement status in real time.</span
					>
				</span>
				<span aria-hidden="true" class="text-3xl text-blue-700 transition group-hover:translate-x-1"
					>→</span
				>
			</span>
		</a>
	{/if}
	{#if data.user.canExportCheckins}
		<a
			href="/staff/checkins"
			class="group w-full max-w-2xl rounded-2xl border border-emerald-200 bg-emerald-50 p-6 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-emerald-400 hover:shadow-md focus:outline-none focus:ring-4 focus:ring-emerald-200 dark:border-emerald-900 dark:bg-emerald-950 dark:focus:ring-emerald-900"
		>
			<span
				class="text-sm font-semibold uppercase tracking-[0.16em] text-emerald-700 dark:text-emerald-300"
				>Restricted data tool</span
			>
			<span class="mt-2 flex items-center justify-between gap-4">
				<span>
					<span class="block text-2xl font-bold text-gray-950 dark:text-white"
						>Check-in Data Export</span
					>
					<span class="mt-1 block text-gray-700 dark:text-gray-200"
						>Download card-tap events with stable, privacy-safe member UUIDs.</span
					>
				</span>
				<span
					aria-hidden="true"
					class="text-3xl text-emerald-700 transition group-hover:translate-x-1">→</span
				>
			</span>
		</a>
	{/if}
	<div class="flex w-full flex-col items-center justify-center">
		<P class="mb-6 text-lg dark:text-gray-400 sm:px-16 lg:text-xl xl:px-48">Staff hours.</P>
		<Calendar url="/staff/calendar" />
	</div>
</div>
