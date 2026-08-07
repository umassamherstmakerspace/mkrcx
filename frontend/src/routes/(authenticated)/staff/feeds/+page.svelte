<script lang="ts">
	import { onMount } from 'svelte';
	import type { LeashFeed } from '$lib/leash';
	import type { PageData } from './$types';

	export let data: PageData;

	let feeds: LeashFeed[] = [];
	let loading = true;
	let failure = '';

	onMount(async () => {
		try {
			feeds = (await data.api.listFeeds({ limit: 100 }, true)).data;
		} catch (error) {
			failure = error instanceof Error ? error.message : 'Unable to load feeds.';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head><title>Live feeds · mkr.cx</title></svelte:head>

<main class="mx-auto flex max-w-5xl flex-col gap-6">
	<div>
		<p class="text-sm font-semibold uppercase tracking-[0.18em] text-blue-600 dark:text-blue-400">
			mkr.cx
		</p>
		<h1 class="text-3xl font-bold text-gray-950 dark:text-white">Live feeds</h1>
		<p class="mt-2 text-gray-600 dark:text-gray-300">Choose a feed to open its heads-up display.</p>
	</div>

	{#if loading}
		<p class="text-gray-500">Loading feeds…</p>
	{:else if failure}
		<div
			class="rounded-xl border border-red-300 bg-red-50 p-4 text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
		>
			{failure}
		</div>
	{:else if feeds.length === 0}
		<div
			class="rounded-xl border border-gray-200 p-8 text-center text-gray-500 dark:border-gray-700"
		>
			No readable feeds are configured.
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each feeds as feed}
				<a
					href={`/staff/feeds/${feed.ID}`}
					class="group rounded-2xl border border-gray-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-blue-400 hover:shadow-md dark:border-gray-700 dark:bg-gray-900"
				>
					<div class="flex items-center justify-between gap-4">
						<h2 class="text-xl font-semibold capitalize text-gray-950 dark:text-white">
							{feed.Name.replaceAll('-', ' ')}
						</h2>
						<span
							aria-hidden="true"
							class="text-2xl text-blue-600 transition group-hover:translate-x-1">→</span
						>
					</div>
					<p class="mt-2 font-mono text-sm text-gray-500">{feed.Name}</p>
				</a>
			{/each}
		</div>
	{/if}
</main>
