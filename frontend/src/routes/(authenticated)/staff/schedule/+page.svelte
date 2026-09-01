<script lang="ts">
	import Calendar from '$lib/components/Calendar.svelte';
	import type { PageData } from './$types';
	import { resolveStaffShiftName } from '$lib/staffShiftExport';

	export let data: PageData;
	$: shiftCalendarName = resolveStaffShiftName(data.user);
</script>

<div class="w-full px-2 pb-12 md:px-6">
	<section aria-labelledby="staff-calendar-heading">
		<header class="mb-5 flex flex-col gap-3 text-left sm:flex-row sm:items-end sm:justify-between">
			<div>
				<h1 id="staff-calendar-heading" class="text-3xl font-bold text-gray-950 dark:text-white">
					Staff calendar
				</h1>
				<p class="mt-1 text-gray-600 dark:text-gray-300">
					Shifts are color-coded; tours and other events are gray. Select any event for its details.
				</p>
			</div>
			{#if shiftCalendarName}
				<a
					href="/staff/shifts.ics"
					download
					class="inline-flex shrink-0 items-center justify-center rounded-lg bg-violet-700 px-4 py-2.5 text-sm font-semibold text-white hover:bg-violet-800 focus:outline-none focus:ring-4 focus:ring-violet-300 dark:focus:ring-violet-900"
				>
					Download my shifts (.ics)
				</a>
			{/if}
		</header>

		<Calendar url="/staff/calendar" detailsEnabled fitWorkingHours />
	</section>
</div>
