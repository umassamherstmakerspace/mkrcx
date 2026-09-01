<script lang="ts">
	import Calendar from '$lib/components/Calendar.svelte';
</script>

<div class="flex w-full flex-col items-center px-2 pb-12 text-center md:px-8">
	<a
		href="/note"
		class="mb-5 flex w-full max-w-4xl items-center justify-between gap-4 rounded-xl border border-red-200 bg-red-50 p-4 text-left transition hover:border-red-400 hover:shadow-sm focus:outline-none focus:ring-4 focus:ring-red-200 dark:border-red-950 dark:bg-red-950 dark:focus:ring-red-950 md:p-5"
	>
		<span>
			<span class="block text-xl font-bold text-gray-950 dark:text-white">Send a note</span>
			<span class="mt-1 block text-sm text-gray-700 dark:text-gray-200 md:text-base"
				>Share an observation, problem, or anything Makerspace staff should know.</span
			>
		</span>
		<span aria-hidden="true" class="text-3xl text-red-800 dark:text-red-200">→</span>
	</a>
	<div class="flex w-full flex-col items-center justify-center">
		<Calendar url="/calendar" fitWorkingHours />
	</div>
</div>
