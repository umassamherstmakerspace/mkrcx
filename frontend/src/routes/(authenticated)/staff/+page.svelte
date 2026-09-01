<script lang="ts">
	import type { PageData } from './$types';

	export let data: PageData;
</script>

<div class="w-full space-y-6 px-2 pb-12 md:px-6">
	<header class="text-left">
		<h1 class="text-3xl font-bold text-gray-950 dark:text-white">Staff Zone</h1>
		<p class="mt-1 text-gray-600 dark:text-gray-300">Choose a staff tool.</p>
	</header>

	<section aria-labelledby="staff-tools-heading">
		<h2 id="staff-tools-heading" class="sr-only">Staff tools</h2>
		<div class="grid gap-3 md:grid-cols-2">
			<a
				href="/staff/schedule"
				class="group flex items-center justify-between gap-4 rounded-xl border border-violet-200 bg-violet-50 p-4 text-left transition hover:border-violet-400 hover:shadow-sm focus:outline-none focus:ring-4 focus:ring-violet-200 dark:border-violet-900 dark:bg-violet-950 dark:focus:ring-violet-900"
			>
				<span>
					<span class="block text-lg font-semibold text-gray-950 dark:text-white"
						>Staff Calendar</span
					>
					<span class="mt-1 block text-sm text-gray-700 dark:text-gray-200"
						>View staff shifts, tours, and other scheduled events.</span
					>
				</span>
				<span
					aria-hidden="true"
					class="text-2xl text-violet-700 transition group-hover:translate-x-1">→</span
				>
			</a>

			{#if data.user.canListFeeds}
				<a
					href="/staff/feeds/1"
					class="group flex items-center justify-between gap-4 rounded-xl border border-blue-200 bg-blue-50 p-4 text-left transition hover:border-blue-400 hover:shadow-sm focus:outline-none focus:ring-4 focus:ring-blue-200 dark:border-blue-900 dark:bg-blue-950 dark:focus:ring-blue-900"
				>
					<span>
						<span class="block text-lg font-semibold text-gray-950 dark:text-white"
							>Front Desk Check-in HUD</span
						>
						<span class="mt-1 block text-sm text-gray-700 dark:text-gray-200"
							>Monitor card taps and participation-agreement status.</span
						>
					</span>
					<span
						aria-hidden="true"
						class="text-2xl text-blue-700 transition group-hover:translate-x-1">→</span
					>
				</a>
			{/if}

			{#if data.user.canExportCheckins}
				<a
					href="/staff/checkins"
					class="group flex items-center justify-between gap-4 rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-left transition hover:border-emerald-400 hover:shadow-sm focus:outline-none focus:ring-4 focus:ring-emerald-200 dark:border-emerald-900 dark:bg-emerald-950 dark:focus:ring-emerald-900"
				>
					<span>
						<span class="block text-lg font-semibold text-gray-950 dark:text-white"
							>Check-in Data Export</span
						>
						<span class="mt-1 block text-sm text-gray-700 dark:text-gray-200"
							>Download privacy-safe card-tap data.</span
						>
					</span>
					<span
						aria-hidden="true"
						class="text-2xl text-emerald-700 transition group-hover:translate-x-1">→</span
					>
				</a>
			{/if}
		</div>
	</section>
</div>
