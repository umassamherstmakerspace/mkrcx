<script lang="ts">
	import Calendar from '$lib/components/Calendar.svelte';
</script>

<div class="flex w-full flex-col items-center px-2 pb-12 text-center md:px-8">
	<a
		href="/note"
		class="mb-5 flex w-full max-w-4xl items-center justify-between gap-4 rounded-xl border border-blue-200 bg-blue-50 p-4 text-left transition hover:border-blue-400 hover:shadow-sm focus:outline-none focus:ring-4 focus:ring-blue-200 dark:border-blue-900 dark:bg-blue-950 dark:focus:ring-blue-900 md:p-5"
	>
		<span>
			<span class="block text-xl font-bold text-gray-950 dark:text-white">Send a note</span>
			<span class="mt-1 block text-sm text-gray-700 dark:text-gray-200 md:text-base"
				>Share an observation with the Makerspace team.</span
			>
		</span>
		<span aria-hidden="true" class="text-3xl text-blue-800 dark:text-blue-200">→</span>
	</a>
	<div class="flex w-full flex-col items-center justify-center">
		<Calendar url="/calendar" fitWorkingHours />
	</div>
</div>
