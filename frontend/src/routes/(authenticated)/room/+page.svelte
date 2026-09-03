<script lang="ts">
	import Calendar from '$lib/components/Calendar.svelte';
	import type { PageData } from './$types';

	export let data: PageData;
</script>

<svelte:head>
	<title>Ag Eng 119 Availability · UMass Makerspace</title>
	<meta
		name="description"
		content="See when the UMass Makerspace room in Agricultural Engineering 119 is available."
	/>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="w-full px-2 pb-12 md:px-6">
	<section aria-labelledby="room-calendar-heading">
		<header class="mb-5 text-left">
			<div class="flex flex-wrap items-center gap-3">
				<h1 id="room-calendar-heading" class="text-3xl font-bold text-gray-950 dark:text-white">
					Ag Eng 119 availability
				</h1>
				<span
					class="rounded border border-gray-200 px-2 py-0.5 text-xs text-gray-500 dark:border-gray-600 dark:text-gray-300"
					>{data.user.isStaff ? 'Staff view' : 'Free/busy view'}</span
				>
			</div>
			<p class="mt-1 text-gray-600 dark:text-gray-300">
				{data.user.isStaff
					? 'Select a booking to see its details.'
					: 'Room bookings are shown as Busy; names and event details are hidden.'}
			</p>
		</header>

		<Calendar url="/room/calendar" detailsEnabled={data.user.isStaff} fitWorkingHours />
	</section>
</div>
