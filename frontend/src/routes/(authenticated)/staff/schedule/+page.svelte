<script lang="ts">
	import Calendar from '$lib/components/Calendar.svelte';
</script>

<div class="w-full px-2 pb-12 md:px-6">
	<section aria-labelledby="staff-calendar-heading">
		<header class="mb-5 text-left">
			<h1 id="staff-calendar-heading" class="text-3xl font-bold text-gray-950 dark:text-white">
				Staff calendar
			</h1>
			<p class="mt-1 text-gray-600 dark:text-gray-300">
				Shifts are color-coded; tours and other events are gray. Select any event for its details.
			</p>
		</header>

		<Calendar url="/staff/calendar" detailsEnabled fitWorkingHours />
	</section>
</div>
